package version

import (
	"context"
	"flag"
	"fmt"
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
	t.Run(fmt.Sprintf("golden-%s_%s", runtime.GOOS, runtime.GOARCH), func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cmd := Command{}

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}
