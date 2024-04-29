package uitest

import (
	"context"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii) // no color for consistent golden file
	lipgloss.SetHasDarkBackground(false)    // dark background for consistent golden file
}

func TestTUI(ctx context.Context, t *testing.T) (*ui.Driver, *teatest.TestModel) {
	drv := ui.NewDriverTest(ctx)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(128, 64))

	drv.Program = program{tm}

	return drv, tm
}

type program struct {
	*teatest.TestModel
}

func (p program) Quit() {
	err := p.TestModel.Quit()
	if err != nil {
		panic(err)
	}
}

func (p program) Run() (tea.Model, error) {
	panic("TODO")
}

func (p program) Wait() {
	// no-op, for TestError and since TestModel doesn't provide a Wait without needing a t.testing
}

func TestTUIError(ctx context.Context, t *testing.T, tui cli.UI, msgAndArgs ...interface{}) {
	_, errc := testTUI(ctx, t, tui)
	err := <-errc
	require.Error(t, err, msgAndArgs...)
}

func TestTUIOutput(ctx context.Context, t *testing.T, tui cli.UI) {
	drv, errc := testTUI(ctx, t, tui)

	out, err := io.ReadAll(drv.Out)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}

func testTUI(ctx context.Context, t *testing.T, tui cli.UI) (ui.Driver, chan error) {
	drv := ui.NewDriverTest(ctx)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(128, 64))

	drv.Program = program{tm}

	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		defer tm.Quit()

		errc <- tui.RunTUI(ctx, drv)
	}()

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second*3))

	return *drv, errc
}
