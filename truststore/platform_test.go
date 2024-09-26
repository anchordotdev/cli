package truststore

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"reflect"
	"testing"
)

func TestParseCertificate(t *testing.T) {
	cert, err := parseCertificate(validCA.Raw)
	if err != nil {
		t.Fatal(err)
	}
	if cert == nil {
		t.Fatal("expect parse certificate with valid certificate to return certificate")
	}

}

func TestParseInvalidCerts(t *testing.T) {
	tests := []struct {
		name string

		data string

		cert *x509.Certificate
		err  error
	}{
		{
			name: "duplicate-extension",

			data: dupExtensionData,
		},
		{
			name: "inner-outer-signature-mismatch",

			data: signatureMismatchData,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blk, _ := pem.Decode([]byte(test.data))
			cert, err := parseCertificate(blk.Bytes)
			if want, got := test.cert, cert; !reflect.DeepEqual(want, got) {
				t.Errorf("expect parsed certificate %+v, got %+v", want, got)
			}
			if want, got := test.err, err; !errors.Is(want, got) {
				t.Errorf("expect err %s, got %s", want, got)
			}
		})
	}
}

var (
	dupExtensionData = `-----BEGIN CERTIFICATE-----
MIIDLDCCAhSgAwIBAgIESJZhgDANBgkqhkiG9w0BAQsFADA7MR8wHQYDVQQDDBZj
b20uYXBwbGUua2VyYmVyb3Mua2RjMRgwFgYDVQQKDA9TeXN0ZW0gSWRlbnRpdHkw
HhcNMTYxMjIxMjIzODE5WhcNMzYxMjE2MjIzODE5WjA7MR8wHQYDVQQDDBZjb20u
YXBwbGUua2VyYmVyb3Mua2RjMRgwFgYDVQQKDA9TeXN0ZW0gSWRlbnRpdHkwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCw+KArez/ODHC5bcIjIhkLMe4d
06gLxOmc9NN1yG3bPSp6I/7kOc26f70YttvK6loPpEyGRTxyyC7RepALi+TtC6Ia
BcwrTQTLVfHND0a0yjZLfTTTFTl76BYVdBoP1Ta6Kh0/+ufAjpUvFOGo3fES0JRt
aFyluO/nFH2GGuwwOl5r3+CRECv2ipJcbHyYllpP6oAS1LK59m2UYj/rlmzD4Kx9
NENwSuoaOBXsg74lQsX5JYgPA/UTv2TfjVLOw6yyncgfwie+nTd4XphxnKpedFWf
iYdZeMJQPzFw47GbtjBrfW8FwKTQbCoosAEFpQ6cwoazKSzt7ICS9J8zur5FAgMB
AAGjODA2MAsGA1UdDwQEAwIFoDATBgNVHSUEDDAKBggrBgEFBQcDATASBgNVHSUE
CzAJBgcrBgEFAgMFMA0GCSqGSIb3DQEBCwUAA4IBAQB57yZrn3PrWMZucdcfOcy2
lzth2iYGZvNQ2YNnmfU3W9Umi8Ku7PVlVAR3DaFQIs99uejlvrjvLrYLyIhklA1f
hyGljDJFHZVceAO0qHA5gjd0p95Z4l+04NqKPcdV4PjSM3RSX9LiWieizJHezloe
gHtW2OUPVe138Ic2OAQNB0e0/0FK6h/B96sYcskvwZF2xnOkjOFJimh5iUPIemtT
Oi3a6RdwSBzfJTtO9bSQ+lGdkmJAQ0XB3REJPIcLDz7QIG8cRXX4yFnjaHw0kM12
ZyvlXrsgZkrum/0zNBWAnp/MEeTPJzsl75Fu2C+qO7IRMeirP4/Jf6+SWy3BxXNz
-----END CERTIFICATE-----`
	signatureMismatchData = `-----BEGIN CERTIFICATE-----
MIICFDCCAX2gAwIBAgIECZcijjALBgkqhkiG9w0BAQUwPDEgMB4GA1UEAwwXY29t
LmFwcGxlLnN5c3RlbWRlZmF1bHQxGDAWBgNVBAoMD1N5c3RlbSBJZGVudGl0eTAe
Fw0xNTAzMjUyMTEwMjZaFw0zNTAzMjAyMTEwMjZaMDwxIDAeBgNVBAMMF2NvbS5h
cHBsZS5zeXN0ZW1kZWZhdWx0MRgwFgYDVQQKDA9TeXN0ZW0gSWRlbnRpdHkwgZ8w
DQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBALonEb9P9qvj1BiLCQp86oLkVC+riuqd
llz1JJEtbr2BFEAa/j0CH+FptthLizHSWsVdHN+CrPZUa1rCgVlkuz14huzSPrQG
UjNjpJVDBmlr/T+ALmHewmykM9Sa3yOETVr8q49odjisfcIHLwS+ivymLDgU3mPJ
Qss2jW+Av6lRAgMBAAGjJTAjMAsGA1UdDwQEAwIEsDAUBgNVHSUEDTALBgkqhkiG
92NkBAQwDQYJKoZIhvcNAQEFBQADgYEAroEWpwSHikgb1zjueWPdXwY4o+W+zFqY
uVbrTzd+Tv8SIfgw8+D4Hf9iLLY33yy6CIMZY2xgfGgBh0suSidoLJt3Pr0fiQGK
d5IUuavJmM5HeYXlPfg/WxvtcwaB1DlPxGpe3ZsRi2GPBZpxVS1AdwKUk5GmoH4G
J1hlJQKJ8yY=
-----END CERTIFICATE-----
`
)
