package truststore

import (
	"testing"
)

func TestParseCertNick(t *testing.T) {
	var tests = []struct {
		name string

		line string

		nick string
	}{
		{
			name: "normal",

			line: "f016d6d279570cd2ac25debd                                     C,,  ",

			nick: "f016d6d279570cd2ac25debd",
		},
		{
			name: "long",

			line: "docert development CA FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF C,,  ",

			nick: "docert development CA FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF",
		},
		{
			name: "very-long",

			line: "Baddy Local Authority - 2024 ECC Root 000000000000000000000000000000000000000 C,,  ",

			nick: "Baddy Local Authority - 2024 ECC Root 000000000000000000000000000000000000000",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.nick, parseCertNick(test.line); want != got {
				t.Errorf("want parsed cert nick %q, got %q", want, got)
			}
		})
	}
}
