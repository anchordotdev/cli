package trust

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ext509"
	"github.com/anchordotdev/cli/internal/must"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestCmdTrustAudit(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdTrustAudit, "trust", "audit", "--help")
	})

	t.Run("--org testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustAudit, "--org", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustAudit, "-o", "testOrg")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [realm]")
	})

	t.Run("-o testOrg -r testRealm", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrustAudit, "-o", "testOrg", "-r", "testRealm")
		require.Equal(t, "testOrg", cfg.Trust.Org)
		require.Equal(t, "testRealm", cfg.Trust.Realm)
	})

	t.Run("--realm testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustAudit, "--realm", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("-r testRealm", func(t *testing.T) {
		err := cmdtest.TestError(t, CmdTrustAudit, "-r", "testRealm")
		require.ErrorContains(t, err, "if any flags in the group [org realm] are set they must all be set; missing [org]")
	})

	t.Run("default --trust-stores", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust)
		require.Equal(t, []string{"homebrew", "nss", "system"}, cfg.Trust.Stores)
	})

	t.Run("--trust-stores nss,system", func(t *testing.T) {
		cfg := cmdtest.TestCfg(t, CmdTrust, "--trust-stores", "nss,system")
		require.Equal(t, []string{"nss", "system"}, cfg.Trust.Stores)
	})
}

func TestAudit(t *testing.T) {
	if srv.IsProxy() {
		t.Skip("trust audit unsupported in proxy mode")
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
	ctx = cli.ContextWithConfig(ctx, cfg)

	anc, err := api.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}

	org, realm, err := fetchOrgAndRealm(ctx, anc)
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

		cmd := &Audit{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
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
