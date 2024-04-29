package cli_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/ui/uitest"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var CmdError = cli.NewCmd[ErrorCommand](nil, "error", func(cmd *cobra.Command) {})

type ErrorCommand struct{}

func (c ErrorCommand) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

var testErr = errors.New("test error")

func (c *ErrorCommand) run(ctx context.Context, drv *ui.Driver) error {
	return testErr
}

func TestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := cli.Config{}
	cfg.NonInteractive = true
	cfg.Test.Browserless = true
	ctx = cli.ContextWithConfig(ctx, &cfg)

	t.Run(fmt.Sprintf("golden-%s_%s", runtime.GOOS, runtime.GOARCH), func(t *testing.T) {
		var returnedError error

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := ErrorCommand{}

		defer func() {
			require.Error(t, returnedError)
			require.EqualError(t, returnedError, testErr.Error())

			tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
			teatest.RequireEqualOutput(t, drv.FinalOut())
		}()
		defer cli.Cleanup(&returnedError, nil, ctx, drv, CmdError, []string{})
		returnedError = cmd.UI().RunTUI(ctx, drv)
	})
}

var CmdPanic = cli.NewCmd[PanicCommand](nil, "error", func(cmd *cobra.Command) {})

type PanicCommand struct{}

func (c PanicCommand) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *PanicCommand) run(ctx context.Context, drv *ui.Driver) error {
	panic("test panic")
}

func TestPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := cli.Config{}
	cfg.NonInteractive = true
	cfg.Test.Browserless = true
	ctx = cli.ContextWithConfig(ctx, &cfg)

	t.Run(fmt.Sprintf("golden-%s_%s", runtime.GOOS, runtime.GOARCH), func(t *testing.T) {
		var returnedError error

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := PanicCommand{}

		defer func() {
			require.Error(t, returnedError)
			require.EqualError(t, returnedError, "test panic")

			tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
			teatest.RequireEqualOutput(t, drv.FinalOut())
		}()
		defer cli.Cleanup(&returnedError, nil, ctx, drv, CmdPanic, []string{})
		_ = cmd.UI().RunTUI(ctx, drv)
	})
}
