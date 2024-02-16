package truststore

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"reflect"
	"testing"

	"github.com/anchordotdev/cli/internal/must"
)

func TestAudit(t *testing.T) {
	MockCAs = []*CA{
		validCA,
		extraCA,
	}
	defer func() { MockCAs = nil }()

	store := new(Mock)

	aud := Audit{
		Expected: []*CA{validCA, missingCA},

		Stores: []Store{store},
	}

	info, err := aud.Perform()
	if err != nil {
		t.Fatal(err)
	}

	if want, got := []*CA{validCA}, info.Valid; !reflect.DeepEqual(want, got) {
		t.Errorf("want valid cas %+v, got %+v", want, got)
	}

	if want, got := []*CA{missingCA}, info.Missing; !reflect.DeepEqual(want, got) {
		t.Errorf("want missing cas %+v, got %+v", want, got)
	}

	if want, got := []*CA{extraCA}, info.Extra; !reflect.DeepEqual(want, got) {
		t.Errorf("want missing cas %+v, got %+v", want, got)
	}

	if !info.IsPresent(validCA, store) {
		t.Errorf("want present ca %+v in store %+v, but was not", validCA, store)
	}
	if !info.IsPresent(extraCA, store) {
		t.Errorf("want extra ca %+v in store %+v, but was not", extraCA, store)
	}

	if info.IsPresent(missingCA, store) {
		t.Errorf("want missing ca %+v not in store %+v, but was", missingCA, store)
	}
}

var (
	validCA = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Valid CA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}))

	missingCA = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Missing CA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}))

	extraCA = mustCA(must.CA(&x509.Certificate{
		Subject: pkix.Name{
			CommonName:   "Extra CA",
			Organization: []string{"Example, Inc"},
		},
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}))
)
