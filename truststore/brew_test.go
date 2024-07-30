//go:build darwin
// +build darwin

package truststore

import (
	"testing"
)

func TestBrew(t *testing.T) {
	testFS := make(TestFS, 1)
	if err := testFS.AppendToFile("cert.pem", nil); err != nil {
		t.Fatal(err)
	}

	store := &Brew{
		RootDir: "/",
		DataFS:  testFS,
		SysFS:   RootFS(),

		certPath: "cert.pem",
	}

	testStore(t, store)
}
