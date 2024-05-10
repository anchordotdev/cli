package cli_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	_ "github.com/anchordotdev/cli/testflags"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/ui/uitest"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func setupCleanup(t *testing.T) {
	t.Helper()

	colorProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)

	cliOS, cliArch := cli.Version.Os, cli.Version.Arch
	cli.Version.Os, cli.Version.Arch = "goos", "goarch"

	t.Cleanup(func() {
		lipgloss.SetColorProfile(colorProfile)

		cli.Version.Os, cli.Version.Arch = cliOS, cliArch
	})
}

func testTag() string {
	switch runtime.GOOS {
	case "darwin", "linux":
		return "unix"
	default:
		return runtime.GOOS
	}
}

var CmdError = cli.NewCmd[ErrorCommand](nil, "error", func(cmd *cobra.Command) {})

type ErrorCommand struct{}

func (c ErrorCommand) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

var testErr = errors.New("test error")

func (c *ErrorCommand) run(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &TestHeader{Type: "error"})
	drv.Activate(ctx, &TestHint{Type: "error"})

	return testErr
}

func TestError(t *testing.T) {
	setupCleanup(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := cli.Config{}
	cfg.NonInteractive = true
	cfg.Test.Browserless = true
	ctx = cli.ContextWithConfig(ctx, &cfg)

	t.Run(fmt.Sprintf("golden-%s", testTag()), func(t *testing.T) {
		var returnedError error

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := ErrorCommand{}

		defer func() {
			require.Error(t, returnedError)
			require.EqualError(t, returnedError, testErr.Error())

			tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
			uitest.TestGolden(t, drv.Golden())
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
	drv.Activate(ctx, &TestHeader{Type: "panic"})
	drv.Activate(ctx, &TestHint{Type: "panic"})
	panic("test panic")
}

func TestPanic(t *testing.T) {
	setupCleanup(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := cli.Config{}
	cfg.NonInteractive = true
	cfg.Test.Browserless = true
	ctx = cli.ContextWithConfig(ctx, &cfg)

	t.Run(fmt.Sprintf("golden-%s", testTag()), func(t *testing.T) {
		var returnedError error

		drv, tm := uitest.TestTUI(ctx, t)

		cmd := PanicCommand{}

		defer func() {
			require.Error(t, returnedError)
			require.EqualError(t, returnedError, "test panic")

			tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
			uitest.TestGolden(t, drv.Golden())
		}()
		defer cli.Cleanup(&returnedError, nil, ctx, drv, CmdPanic, []string{})
		_ = cmd.UI().RunTUI(ctx, drv)
	})
}

type TestHeader struct {
	Type string
}

func (m *TestHeader) Init() tea.Cmd { return nil }

func (m *TestHeader) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TestHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Test %s %s", m.Type, ui.Whisper(fmt.Sprintf("`anchor test %s`", m.Type)))))
	return b.String()
}

type TestHint struct {
	Type string
}

func (m *TestHint) Init() tea.Cmd { return nil }

func (m *TestHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TestHint) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("Test %s Hint.", m.Type)))
	return b.String()
}
