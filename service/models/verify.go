package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	VerifyHeader = ui.Section{
		Name: "VerifyHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Verify Service TLS Setup and Configuration %s", ui.Whisper("`anchor service verify`"))),
		},
	}

	VerifyHint = ui.Section{
		Name: "VerifyHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll check your running app to ensure TLS works as expected."),
		},
	}
)

type Checker struct {
	Name string

	err      error
	finished bool

	spinner spinner.Model
}

func (m *Checker) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type checkerMsg struct {
	mdl *Checker
	err error
}

func (m *Checker) Pass() tea.Msg {
	return checkerMsg{
		mdl: m,
		err: nil,
	}
}

func (m *Checker) Fail(err error) tea.Msg {
	return checkerMsg{
		mdl: m,
		err: err,
	}
}

func (m *Checker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case checkerMsg:
		if msg.mdl == m {
			m.finished = true
			m.err = msg.err
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *Checker) View() string {
	var b strings.Builder
	switch {
	case !m.finished:
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Checking %s…%s",
			m.Name,
			m.spinner.View())))
	case m.err == nil:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Checked %s: success!",
			m.Name)))
	default:
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Checked %s: failed!",
			m.Name)))
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Error! %s",
			m.err.Error())))
	}
	return b.String()
}
