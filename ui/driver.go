package ui

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

var (
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

	finalOut bytes.Buffer
	Out      io.Reader
	out      io.ReadWriter
	test     bool
	LastView string
}

func NewDriverTest(ctx context.Context) *Driver {
	drv := new(Driver)

	opts := []tea.ProgramOption{
		tea.WithInputTTY(),
		tea.WithContext(ctx),
		tea.WithoutRenderer(),
	}
	drv.Program = tea.NewProgram(drv, opts...)

	drv.out = &safeReadWriter{rw: new(bytes.Buffer)}
	drv.Out = io.TeeReader(drv.out, &drv.finalOut)
	drv.test = true

	return drv
}

func NewDriverTUI(ctx context.Context) (*Driver, Program) {
	drv := new(Driver)

	opts := []tea.ProgramOption{
		tea.WithInputTTY(),
		tea.WithContext(ctx),
	}

	drv.out = &safeReadWriter{rw: new(bytes.Buffer)}
	drv.Out = io.TeeReader(drv.out, &drv.finalOut)
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

func (d *Driver) FinalOut() []byte {
	io.ReadAll(d.Out)
	return d.finalOut.Bytes()
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

func (d *Driver) View() string {
	var out string
	for _, mdl := range d.models {
		out += mdl.View()
	}
	normalizedOut := spinnerReplacer.Replace(out)
	if out != "" && normalizedOut != d.LastView {
		separator := fmt.Sprintf("─── %s ", reflect.TypeOf(d.active).Elem().Name())
		separator = separator + strings.Repeat("─", 80-utf8.RuneCountInString(separator))
		fmt.Fprintln(d.out, separator)
		fmt.Fprint(d.out, normalizedOut)
		d.LastView = normalizedOut
	}
	return out
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

// safeReadWriter implements io.ReadWriter, but locks reads and writes.
type safeReadWriter struct {
	sync.RWMutex

	rw io.ReadWriter
}

// Read implements io.ReadWriter.
func (s *safeReadWriter) Read(p []byte) (n int, err error) {
	s.RLock()
	defer s.RUnlock()
	return s.rw.Read(p) //nolint: wrapcheck
}

// Write implements io.ReadWriter.
func (s *safeReadWriter) Write(p []byte) (int, error) {
	s.Lock()
	defer s.Unlock()
	return s.rw.Write(p) //nolint: wrapcheck
}
