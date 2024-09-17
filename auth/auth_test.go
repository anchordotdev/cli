package auth

import (
	"context"
	"os"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
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

func TestCmdAuth(t *testing.T) {
	t.Run("auth", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuth, "auth")
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdAuth, "auth", "--help")
	})
}
