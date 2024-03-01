package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type AuditUnauthenticated bool

type AuditHeader struct{}

func (AuditHeader) Init() tea.Cmd { return nil }

func (m *AuditHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *AuditHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Audit lcl.host HTTPS Local Development Environment %s", ui.Whisper("`anchor lcl audit`"))))
	return b.String()
}

type AuditHint struct{}

func (AuditHint) Init() tea.Cmd { return nil }

func (m *AuditHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *AuditHint) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepHint("We'll begin by checking your system to determine what you need for your setup."))
	return b.String()
}

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

	fmt.Fprintln(&b, ui.StepDone("Checked resources on Anchor.dev: found provisioned resources."))
	return b.String()
}

type AuditTrustMissingMsg int

type AuditTrust struct {
	spinner spinner.Model

	finished bool
	missing  int
}

func (m *AuditTrust) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *AuditTrust) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AuditTrustMissingMsg:
		m.finished = true
		m.missing = int(msg)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *AuditTrust) View() string {
	var b strings.Builder

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Scanning local and expected CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	if m.missing > 0 {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Scanned local and expected CA certificates: need to install %d missing certificates.", m.missing)))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Scanned local and expected CA certificates: found matching certificates."))
	return b.String()
}
