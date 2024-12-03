package org

import (
	"context"
	"os"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
	_ "github.com/anchordotdev/cli/testflags"
)

var srv = &apitest.Server{
	Host:    "api.anchor.lcl.host",
	RootDir: "../..",
}

func TestMain(m *testing.M) {
	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}

	defer os.Exit(m.Run())

	srv.Close()
}
