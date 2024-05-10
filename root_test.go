package cli_test

import (
	"testing"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cmdtest"
	_ "github.com/anchordotdev/cli/lcl"
	_ "github.com/anchordotdev/cli/testflags"
	_ "github.com/anchordotdev/cli/trust"
	_ "github.com/anchordotdev/cli/version"
)

func TestCmdRoot(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		cmdtest.TestHelp(t, cli.CmdRoot)
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, cli.CmdRoot, "--help")
	})
}
