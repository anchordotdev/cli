package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type AuditUnauthenticated bool

var (
	AuditHeader = ui.Section{
		Name: "AuditHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Audit lcl.host HTTPS Local Development Environment %s", ui.Whisper("`anchor lcl audit`"))),
		},
	}

	AuditHint = ui.Section{
		Name: "AuditHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll determine what needs setup on your system."),
		},
	}
)

type AuditResourcesFoundMsg struct{}
type AuditResourcesNotFoundMsg struct{}

type AuditResources struct {
	spinner spinner.Model

	found, notFound bool
}

func (m *AuditResources) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *AuditResources) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AuditResourcesFoundMsg:
		m.found = true
		return m, nil
	case AuditResourcesNotFoundMsg:
		m.notFound = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *AuditResources) View() string {
	var b strings.Builder
	if m.notFound {
		fmt.Fprintln(&b, ui.StepDone("Checked resources on Anchor.dev: need to provision resources."))
		return b.String()
	}

	if !m.found {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Checking resources on Anchor.dev…%s", m.spinner.View())))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Checked resources on Anchor.dev: no provisioning needed."))
	return b.String()
}
