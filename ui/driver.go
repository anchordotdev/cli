package ui

import (
	"context"
	"reflect"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

type Program interface {
	Quit()
	Run() (tea.Model, error)
	Send(tea.Msg)
}

type Driver struct {
	Program // TUI mode

	TTY termenv.File

	models []tea.Model
	active tea.Model
}

func NewDriverTUI(ctx context.Context) (*Driver, Program) {
	drv := new(Driver)

	opts := []tea.ProgramOption{
		tea.WithInputTTY(),
		tea.WithContext(ctx),
		tea.WithoutCatchPanics(), // TODO: remove
	}
	drv.Program = tea.NewProgram(drv, opts...)

	return drv, drv.Program
}

func NewDriverTTY(ctx context.Context) *Driver {
	return &Driver{
		TTY: termenv.DefaultOutput().TTY(),
	}
}

type activateMsg struct {
	tea.Model

	donec chan<- struct{}
}

func (d *Driver) Activate(ctx context.Context, model tea.Model) {
	donec := make(chan struct{})

	d.Send(activateMsg{
		Model: model,
		donec: donec,
	})

	select {
	case <-donec:
	case <-ctx.Done():
	}
}

type stopMsg struct{}

func (d *Driver) Stop() { d.Send(stopMsg{}) }

type pauseMsg chan chan struct{}

func (d *Driver) Pause() chan<- struct{} {
	unpausec := make(chan struct{})
	pausedc := make(chan chan struct{})

	d.Send(pauseMsg(pausedc))

	pausedc <- unpausec

	return unpausec
}

func (d *Driver) Init() tea.Cmd {
	return nil
}

func (d *Driver) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case activateMsg:
		d.models = append(d.models, msg.Model)
		d.active = msg.Model

		close(msg.donec)

		return d, d.active.Init()
	case pauseMsg:
		unpausec := <-msg
		<-unpausec
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return d, tea.Quit
		}
	}

	if d.active == nil {
		if cmd, ok := msg.(tea.Cmd); ok && isQuit(cmd) {
			return d, tea.Quit
		}
		return d, nil
	}

	_, cmd := d.active.Update(msg)
	if isQuit(cmd) {
		d.active = nil
		return d, nil
	}
	return d, cmd
}

func (d *Driver) View() string {
	var out string
	for _, mdl := range d.models {
		out += mdl.View()
	}
	return out
}

var quitPtr = reflect.ValueOf(tea.Quit).Pointer()

func isQuit(cmd tea.Cmd) bool {
	return reflect.ValueOf(cmd).Pointer() == quitPtr
}
