package auth

import (
	"context"
	"flag"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
)

var srv = &apitest.Server{
	Host:    "api.anchor.lcl.host",
	RootDir: "../..",
}

func TestMain(m *testing.M) {
	flag.Parse()

	if err := srv.Start(context.Background()); err != nil {
		panic(err)
	}
	defer srv.Close()

	m.Run()
}

func TestCmdAuth(t *testing.T) {
	t.Run("auth", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuth, "auth")
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuth, "auth", "--help")
	})
}
