package cli_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/stacktrace"
	_ "github.com/anchordotdev/cli/testflags"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/ui/uitest"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"
)

func setupCleanup(t *testing.T) {
	t.Helper()

	cliOS, cliArch := cli.Version.Os, cli.Version.Arch
	cli.Version.Os, cli.Version.Arch = "goos", "goarch"

	cliExecutable := cli.Executable
	switch runtime.GOOS {
	case "darwin", "linux":
		cli.Executable = "/tmp/go-build0123456789/b001/exe/anchor"
	case "windows":
		cli.Executable = `C:\Users\username\AppData\Local\Temp\go-build0123456789/b001/exe/anchor.exe`
	}

	t.Cleanup(func() {
		cli.Version.Os, cli.Version.Arch = cliOS, cliArch
		cli.Executable = cliExecutable
	})
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

	ctx = cli.ContextWithConfig(ctx, &cli.Config{
		NonInteractive: true,
		Test: cli.ConfigTest{
			Browserless: true,
			Timestamp:   Timestamp,
		},
	})

	t.Run(fmt.Sprintf("golden-%s", uitest.TestTagOS()), func(t *testing.T) {
		drv, tm := uitest.TestTUI(ctx, t)

		cmd := ErrorCommand{}
		err := stacktrace.CapturePanic(func() error { return cmd.UI().RunTUI(ctx, drv) })

		if want, got := testErr, err; !reflect.DeepEqual(want, got) {
			t.Fatalf("want return err %+v, got %+v", want, got)
		}

		cli.ReportError(ctx, err, drv, CmdError, nil)

		drv.Program.Quit()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
	})
}

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

	ctx = cli.ContextWithConfig(ctx, &cli.Config{
		NonInteractive: true,
		Test: cli.ConfigTest{
			Browserless: true,
			Timestamp:   Timestamp,
		},
	})

	t.Run(fmt.Sprintf("golden-%s", uitest.TestTagOS()), func(t *testing.T) {
		drv, tm := uitest.TestTUI(ctx, t)

		cmd := PanicCommand{}
		err := stacktrace.CapturePanic(func() error { return cmd.UI().RunTUI(ctx, drv) })

		if want, got := "test panic", err.Error(); want != got {
			t.Fatalf("want return err %q, got %q", want, got)
		}

		cli.ReportError(ctx, err, drv, CmdError, nil)

		drv.Program.Quit()

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))
		uitest.TestGolden(t, drv.Golden())
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

var Timestamp, _ = time.Parse(time.RFC3339Nano, "2024-01-02T15:04:05.987654321Z")
