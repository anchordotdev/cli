package cli_test

import (
	"flag"
	"testing"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cmdtest"
	_ "github.com/anchordotdev/cli/lcl"
	_ "github.com/anchordotdev/cli/trust"
	_ "github.com/anchordotdev/cli/version"
)

var (
	_ = flag.Bool("prism-verbose", false, "ignored")
	_ = flag.Bool("prism-proxy", false, "ignored")
)

func TestCmdRoot(t *testing.T) {
	root := cli.CmdRoot

	t.Run("root", func(t *testing.T) {
		cmdtest.TestOutput(t, root)
	})

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, root, "--help")
	})
}
