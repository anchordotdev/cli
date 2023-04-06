package auth

import (
	"context"
	"flag"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
)

var (
	srv = &apitest.Server{
		RootDir: "../..",
		// RootDir: "../..",
	}

	proxyMode = flag.Bool("proxy", false, "run prism in proxy mode")
)

func TestMain(m *testing.M) {
	flag.Parse()

	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}
	defer srv.Close()

	m.Run()
}
