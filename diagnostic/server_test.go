package diagnostic

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"net/http"
	"testing"

	"github.com/anchordotdev/cli/internal/must"
	_ "github.com/anchordotdev/cli/testflags"
)

func TestServerSupportsDualProtocols(t *testing.T) {
	cert := leaf.TLS()

	srv := &Server{
		Addr: ":0",
		GetCertificate: func(cii *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return &cert, nil
		},
	}

	if err := srv.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	srv.EnableTLS()

	_, port, err := net.SplitHostPort(srv.Addr)
	if err != nil {
		t.Fatal(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: true,
			DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
				return new(net.Dialer).DialContext(ctx, network, srv.Addr)
			},
			TLSClientConfig: &tls.Config{
				RootCAs: anchorCA.CertPool(),
			},
		},
	}

	resHTTP, err := client.Get("http://example.lcl.host.test:" + port)
	if err != nil {
		t.Fatal(err)
	}
	if want, got := http.StatusOK, resHTTP.StatusCode; want != got {
		t.Errorf("want http response status %d, got %d", want, got)
	}
	if got := resHTTP.TLS; got != nil {
		t.Errorf("want nil http response tls info, got %#v", got)
	}

	resHTTPS, err := client.Get("https://example.lcl.host.test:" + port)
	if err != nil {
		t.Fatal(err)
	}
	if want, got := http.StatusOK, resHTTPS.StatusCode; want != got {
		t.Errorf("want https response status %d, got %d", want, got)
	}
	if got := resHTTPS.TLS; got == nil {
		t.Error("https response tls info was nil")
	}
}

var (
	anchorCA = must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Example CA - AnchorCA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign,
		IsCA:     true,
	})

	subCA = anchorCA.Issue(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Example CA - SubCA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	})

	leaf = subCA.Issue(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "example.lcl.host.test",
			Organization: []string{"Example, Inc"},
		},

		DNSNames: []string{"example.lcl.host.test", "*.example.lcl.host.test"},
	})
)
