//go:build darwin
// +build darwin

package truststore

import (
	"testing"
)

func TestBrew(t *testing.T) {
	testFS := make(TestFS, 1)
	testFS.AppendToFile("cert.pem", nil)

	store := &Brew{
		RootDir: "/",
		DataFS:  testFS,
		SysFS:   RootFS(),

		certPath: "cert.pem",
	}

	testStore(t, store)
}
