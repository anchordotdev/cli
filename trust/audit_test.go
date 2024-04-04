package trust

import (
	"bufio"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"regexp"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/ext509"
	"github.com/anchordotdev/cli/internal/must"
	"github.com/anchordotdev/cli/truststore"
)

func TestAudit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty support on windows")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Trust.Stores = []string{"mock"}

	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}

	anc, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	org, realm, err := fetchOrgAndRealm(ctx, cfg, anc)
	if err != nil {
		t.Fatal(err)
	}

	expectedCAs, err := fetchExpectedCAs(ctx, anc, org, realm)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("expected, missing, and extra CAs", func(t *testing.T) {
		truststore.MockCAs = []*truststore.CA{
			expectedCAs[0],
			extraCA,
			ignoredCA,
		}
		defer func() { truststore.MockCAs = nil }()

		cmd := &Audit{
			Config: cfg,
		}

		buf, err := apitest.RunTTY(ctx, cmd.UI())
		if err != nil {
			t.Fatal(err)
		}

		scanner := bufio.NewScanner(buf)

		testOutput(t, scanner, []any{
			regexp.MustCompile(`^VALID    [(][0-9- ]+[)] (RSA|ECDSA)\s+"[a-zA-Z0-9- \/]+ - AnchorCA"$`),
			nil,
			regexp.MustCompile(`\s+Mock\s+TRUSTED$`),
			nil,
			regexp.MustCompile(`^MISSING  [(][0-9- ]+[)] (RSA|ECDSA)\s+"[a-zA-Z0-9- \/]+ - AnchorCA"$`),
			nil,
			regexp.MustCompile(`\s+Mock\s+NOT PRESENT$`),
			nil,
			regexp.MustCompile(`^EXTRA    [(][0-9- ]+[)] (RSA|ECDSA)\s+"Extra CA - AnchorCA"$`),
			nil,
			regexp.MustCompile(`\s+Mock\s+TRUSTED$`),
			nil,
		})
	})
}

func testOutput(t *testing.T, scanner *bufio.Scanner, lines []any) {
	t.Helper()

	for _, line := range lines {
		if !scanner.Scan() {
			t.Fatalf("want more output, got %q %v (nil is EOF)", scanner.Err(), scanner.Err())
		}
		switch line := line.(type) {
		case string:
			if want, got := line, scanner.Text(); want != got {
				t.Errorf("want output line %q, got %q", want, got)
			}
		case *regexp.Regexp:
			if want, got := line, scanner.Text(); !want.MatchString(got) {
				t.Errorf("want output line %q to match %q", got, want)
			}
		}
	}

	if scanner.Scan() {
		t.Errorf("want EOF, got %q", scanner.Text())
	}
}

var (
	extraCA = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Extra CA - AnchorCA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,

		ExtraExtensions: []pkix.Extension{
			mustAnchorExtension(ext509.AnchorCertificate{}),
		},
	}))

	ignoredCA = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Extra CA - AnchorCA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}))
)

func mustCA(cert *must.Certificate) *truststore.CA {
	uniqueName := cert.Leaf.SerialNumber.Text(16)

	return &truststore.CA{
		Certificate: cert.Leaf,
		FilePath:    "example-ca-" + uniqueName + ".pem",
		UniqueName:  uniqueName,
	}
}

func mustAnchorExtension(anc ext509.AnchorCertificate) pkix.Extension {
	ext, err := anc.Extension()
	if err != nil {
		panic(err)
	}
	return ext
}
