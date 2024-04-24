package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type WhoAmIHeader struct{}

func (m *WhoAmIHeader) Init() tea.Cmd { return nil }

func (m *WhoAmIHeader) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *WhoAmIHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Identify Current Anchor.dev Account %s", ui.Whisper("`anchor auth whoami`"))))
	return b.String()
}

type WhoAmIChecker struct {
	whoami string

	spinner spinner.Model
}

func (m *WhoAmIChecker) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type UserWhoAmIMsg string

func (m *WhoAmIChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case UserWhoAmIMsg:
		m.whoami = string(msg)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *WhoAmIChecker) View() string {
	var b strings.Builder
	if m.whoami == "" {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Identifying Anchor.dev account… %s", m.spinner.View())))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Identified Anchor.dev account: %s", ui.Emphasize(m.whoami))))
	}
	return b.String()
}
