package ui

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

var (
	// regex from https://github.com/acarl005/stripansi
	reAnsiStripper = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

	spinnerReplacer = strings.NewReplacer(strings.Split(spinnerCharacters, ",")...)

	waitingFrames     = WaitingSpinner().Spinner.Frames
	replacementFrame  = waitingFrames[0]
	spinnerCharacters = strings.Join(waitingFrames, ","+replacementFrame+",") + "," + replacementFrame
)

type Program interface {
	Quit()
	Run() (tea.Model, error)
	Send(tea.Msg)
	Wait()
}

type Driver struct {
	Program // TUI mode

	TTY termenv.File

	models []tea.Model
	active tea.Model

	goldenMutex  sync.Mutex
	golden       string
	lastView     string
	replacements []string
}

func NewDriverTest(ctx context.Context) *Driver {
	drv := new(Driver)

	opts := []tea.ProgramOption{
		tea.WithInputTTY(),
		tea.WithContext(ctx),
		tea.WithoutRenderer(),
	}

	drv.Program = tea.NewProgram(drv, opts...)

	return drv
}

func NewDriverTUI(ctx context.Context) (*Driver, Program) {
	drv := new(Driver)

	opts := []tea.ProgramOption{
		tea.WithInputTTY(),
		tea.WithContext(ctx),
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

func (d *Driver) Golden() string {
	d.goldenMutex.Lock()
	defer d.goldenMutex.Unlock()
	return d.golden
}

func (d *Driver) Replace(unsafe, safe string) {
	d.goldenMutex.Lock()
	defer d.goldenMutex.Unlock()

	d.replacements = append(d.replacements, unsafe, safe)
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
	if isExit(cmd) {
		d.active = nil
		return d, tea.Quit
	}
	if isQuit(cmd) {
		d.active = nil
		return d, nil
	}
	return d, cmd
}

func (d *Driver) ErrorView() string {
	var out string
	for _, mdl := range d.models {
		out += mdl.View()
	}
	normalizedOut := spinnerReplacer.Replace(out)
	normalizedOut = reAnsiStripper.ReplaceAllString(normalizedOut, "")
	return normalizedOut
}

func (d *Driver) View() string {
	var out string
	for _, mdl := range d.models {
		out += mdl.View()
	}

	if out != "" {
		d.recordGolden(d.replaced(out))
	}

	return out
}

func (d *Driver) replaced(out string) string {
	replaced := spinnerReplacer.Replace(out)
	replaced = strings.NewReplacer(d.replacements...).Replace(replaced)
	return replaced
}

func (d *Driver) recordGolden(out string) {
	d.goldenMutex.Lock()
	defer d.goldenMutex.Unlock()

	if out == d.lastView {
		return
	}

	var section string
	if mdl, ok := d.active.(interface{ Section() string }); ok {
		section = mdl.Section()
	} else if kind := reflect.TypeOf(d.active).Kind(); kind == reflect.Interface || kind == reflect.Pointer {
		section = reflect.TypeOf(d.active).Elem().Name()
	} else {
		section = reflect.TypeOf(d.active).Name()
	}

	separator := fmt.Sprintf("─── %s ", section)
	if separatorRuneCount := utf8.RuneCountInString(separator); separatorRuneCount < 80 {
		separator = separator + strings.Repeat("─", 80-utf8.RuneCountInString(separator))
	} else {
		runes := []rune(separator)
		separator = string(runes[:79]) + "…"
	}

	d.lastView = out
	d.golden += strings.Join([]string{separator, out}, "\n")
}

var (
	quitPtr = reflect.ValueOf(tea.Quit).Pointer()
	exitPtr = reflect.ValueOf(Exit).Pointer()
)

func isQuit(cmd tea.Cmd) bool {
	return reflect.ValueOf(cmd).Pointer() == quitPtr
}

func isExit(cmd tea.Cmd) bool {
	return reflect.ValueOf(cmd).Pointer() == exitPtr
}

func Exit() tea.Msg { return io.EOF }
