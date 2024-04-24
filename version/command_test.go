package version

import (
	"context"
	"flag"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

var (
	_ = flag.Bool("prism-verbose", false, "ignored")
	_ = flag.Bool("prism-proxy", false, "ignored")
)

func TestCmdVersion(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdVersion, "version", "--help")
	})
}

func TestCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only run version test on linux since OS is in output")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := new(Command)

	uitest.TestTUIOutput(ctx, t, cmd.UI())
}
