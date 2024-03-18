package uitest

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii) // no color for consistent golden file
	lipgloss.SetHasDarkBackground(false)    // dark background for consistent golden file

	ui.Waiting = spinner.Spinner{
		Frames: []string{"*"},
		FPS:    time.Second / 100,
	}
}

func TestTUI(ctx context.Context, t *testing.T) (*ui.Driver, *teatest.TestModel) {
	drv := new(ui.Driver)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(128, 64))

	drv.Program = program{tm}

	return drv, tm
}

type program struct {
	*teatest.TestModel
}

func (p program) Quit() {
	panic("TODO")
}

func (p program) Run() (tea.Model, error) {
	panic("TODO")
}

func TestTUIOutput(ctx context.Context, t *testing.T, tui cli.UI) {
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
	out, err := io.ReadAll(drv.Out)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-errc; err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}
