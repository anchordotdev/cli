package uitest

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/anchordotdev/cli/ui"
)

func TestTUI(ctx context.Context, t *testing.T) (*ui.Driver, *teatest.TestModel) {
	drv := new(ui.Driver)
	tm := teatest.NewTestModel(t, drv, teatest.WithInitialTermSize(800, 600))

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
