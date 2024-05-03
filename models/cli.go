package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

type ReportError struct {
	ConfirmCh chan<- struct{}

	Cmd  *cobra.Command
	Args []string
	Msg  any
}

func (m *ReportError) Init() tea.Cmd { return nil }

func (m *ReportError) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.ConfirmCh != nil {
				close(m.ConfirmCh)
				m.ConfirmCh = nil
			}
		}
	}

	return m, nil
}

func (m *ReportError) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("%s %s %s",
		ui.Error("Error!"),
		m.Msg,
		ui.Whisper(fmt.Sprintf("`%s`", m.Cmd.CalledAs())),
	)))

	fmt.Fprintln(&b, ui.StepHint("We are sorry you encountered this error."))

	if m.ConfirmCh != nil {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to open an issue on Github.",
			ui.Action("Press Enter"),
		)))
	} else {
		fmt.Fprintln(&b, ui.StepDone("Opened an issue on Github."))
	}

	return b.String()
}

type Browserless struct {
	Url string
}

func (m *Browserless) Init() tea.Cmd { return nil }

func (m *Browserless) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *Browserless) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Warning("Unable to open browser."))
	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s this in a browser to continue: %s.",
		ui.Action("Open"),
		ui.URL(m.Url),
	)))

	return b.String()
}
