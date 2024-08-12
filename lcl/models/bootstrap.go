package models

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	BootstrapSkip = ui.Section{
		Name: "BootstrapSkip",
		Model: ui.MessageLines{
			ui.Skip("Initial System Configuration for lcl.host Local HTTPS Development `anchor lcl bootstrap`"),
		},
	}

	BootstrapHeader = ui.Section{
		Name: "BootstrapHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Initial System Configuration for lcl.host Local HTTPS Development %s", ui.Whisper("`anchor lcl bootstrap`"))),
		},
	}

	BootstrapHint = ui.Section{
		Name: "BootstrapHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll configure your browsers and OS to trust your local development certificates."),
		},
	}

	Browserless = ui.Section{
		Name: "Browserless",
		Model: ui.MessageLines{
			ui.Warning("Unable to open browser, skipping browser-based verification."),
		},
	}
)

type Bootstrap struct {
	ConfirmCh chan<- struct{}

	Domain, Port, Scheme string
	ShowHeader           bool

	confirmCh chan<- struct{}
	url       string
}

func (Bootstrap) Init() tea.Cmd { return nil }

type OpenURLMsg string

func (m *Bootstrap) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OpenURLMsg:
		if m.url == "" {
			m.url = string(msg)
			m.confirmCh = m.ConfirmCh
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.confirmCh != nil {
				close(m.confirmCh)
				m.confirmCh = nil
			}
		}
	}

	return m, nil
}

type BootstrapDiagnosticFoundMsg struct{}
type BootstrapDiagnosticNotFoundMsg struct{}

type BootstrapDiagnostic struct {
	spinner spinner.Model

	found, notFound bool
}

func (m *BootstrapDiagnostic) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *BootstrapDiagnostic) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case BootstrapDiagnosticFoundMsg:
		m.found = true
		return m, nil
	case BootstrapDiagnosticNotFoundMsg:
		m.notFound = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *BootstrapDiagnostic) View() string {
	var b strings.Builder
	if m.notFound {
		fmt.Fprintln(&b, ui.StepDone("Checked diagnostic service on Anchor.dev: need to provision service."))
		return b.String()
	}

	if !m.found {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Checking diagnostic service on Anchor.dev…%s", m.spinner.View())))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Checked diagnostic service on Anchor.dev: no provisioning needed."))
	return b.String()
}

func (m Bootstrap) View() string {
	var b strings.Builder

	if m.url == "" {
		return b.String()
	}

	if m.Scheme == "https" {
		fmt.Fprintln(&b, ui.StepHint("Before we move on, let's test HTTPS."))
	}

	if m.confirmCh != nil {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to open %s in your browser.",
			ui.Action("Press Enter"),
			ui.URL(m.url))))

		return b.String()
	}

	if m.Scheme == "http" {
		schemeMessage := "without HTTPS"
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Great, %s works as expected (%s).",
			ui.URL(m.url),
			ui.Accentuate(schemeMessage),
		)))

		fmt.Fprintln(&b, ui.StepHint("Now, we'll add your personal CA certificates to your system's trust stores."))

		return b.String()
	}

	return b.String()
}

type BootstrapSuccess struct {
	URL *url.URL
}

func (BootstrapSuccess) Init() tea.Cmd { return nil }

func (m BootstrapSuccess) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m BootstrapSuccess) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Success! %s works as expected (%s).",
		ui.URL(m.URL.String()),
		ui.Accentuate("encrypted with HTTPS"),
	)))

	return b.String()
}
