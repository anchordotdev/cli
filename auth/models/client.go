package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type AuditAuthenticationWhoami string

type ClientProbed bool
type ClientTested bool

type Client struct {
	spinner spinner.Model

	probed bool
	tested bool
}

func (m *Client) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *Client) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ClientProbed:
		m.probed = bool(msg)
		return m, nil
	case ClientTested:
		m.tested = bool(msg)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *Client) View() string {
	var b strings.Builder

	if !m.probed {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Checking authentication: probing credentials locally…%s", m.spinner.View())))
		return b.String()
	}

	if !m.tested {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Checking authentication: testing credentials remotely…%s", m.spinner.View())))
		return b.String()
	}

	return b.String()
}
