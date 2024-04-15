package auth

import (
	"context"
	"flag"
	"testing"

	"github.com/anchordotdev/cli"
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
	cmd := CmdAuth
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("auth", func(t *testing.T) {
		cfg.Test.SkipRunE = false
		cmdtest.TestOutput(t, cmd, "auth")
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "auth", "--help")
	})
}
