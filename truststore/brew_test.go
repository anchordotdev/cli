package truststore

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"testing"

	"github.com/anchordotdev/cli/truststore/internal/must"
)

var (
	_ = flag.Bool("prism-verbose", false, "ignored")
	_ = flag.Bool("prism-proxy", false, "ignored")
)

func TestBrewCheck(t *testing.T) {
	t.Skip("pending mock filesystem")

	brew := &Brew{
		RootDir: "/",
		DataFS:  RootFS(),
		SysFS:   RootFS(),
	}

	ok, err := brew.Check()
	if err != nil {
		t.Fatal(err)
	}
	if want, got := true, ok; want != got {
		t.Errorf("want check %t, got %t", want, got)
	}

	if ok, err = brew.CheckCA(ca); err != nil {
		t.Fatal(err)
	}
	if want, got := false, ok; want != got {
		t.Errorf("want check ca %t, got %t", want, got)
	}

	if ok, err = brew.InstallCA(ca); err != nil {
		t.Fatal(err)
	}
	if want, got := true, ok; want != got {
		t.Errorf("want install ca %t, got %t", want, got)
	}

	if ok, err = brew.CheckCA(ca); err != nil {
		t.Fatal(err)
	}
	if want, got := true, ok; want != got {
		t.Errorf("want check ca %t, got %t", want, got)
	}
}

var (
	ca = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Example CA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,

		ExtraExtensions: []pkix.Extension{},
	}))
)

func mustCA(cert *must.Certificate) *CA {
	uniqueName := cert.Leaf.SerialNumber.Text(16)

	return &CA{
		Certificate: cert.Leaf,
		FilePath:    "example-ca-" + uniqueName + ".pem",
		UniqueName:  uniqueName,
	}
}
