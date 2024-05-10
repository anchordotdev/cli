package version

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli/cmdtest"
	_ "github.com/anchordotdev/cli/testflags"
	"github.com/anchordotdev/cli/ui/uitest"
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
