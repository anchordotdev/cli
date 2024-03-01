package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type LclSignInHint struct{}

func (LclSignInHint) Init() tea.Cmd { return nil }

func (m *LclSignInHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *LclSignInHint) View() string {
	var b strings.Builder
	// FIXME: first line duplicated from SignInHint, should dedup somehow
	fmt.Fprintln(&b, ui.StepHint("Please sign up or sign in with your Anchor account."))
	fmt.Fprintln(&b, ui.StepHint(""))
	fmt.Fprintln(&b, ui.StepHint("Once authenticated, we can provision your personalized Anchor resources to"))
	fmt.Fprintln(&b, ui.StepHint("power HTTPS in your local development environment."))
	return b.String()
}

type LclPreamble struct{}

func (LclPreamble) Init() tea.Cmd { return nil }

func (m LclPreamble) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclPreamble) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint("Let's set up lcl.host HTTPS in your local development environment!"))
	fmt.Fprintln(&b, ui.Hint(""))
	fmt.Fprintln(&b, ui.Hint("lcl.host (made by the team at Anchor) adds HTTPS in a fast and totally free way"))
	fmt.Fprintln(&b, ui.Hint("to local applications & services."))
	return b.String()
}

type LclHeader struct{}

func (LclHeader) Init() tea.Cmd { return nil }

func (m LclHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Setup lcl.host HTTPS Local Development Environment %s", ui.Whisper("`anchor lcl`"))))
	return b.String()
}

type LclHint struct{}

func (LclHint) Init() tea.Cmd { return nil }

func (m LclHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclHint) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepHint("Once setup finishes, you'll have a secure context in your browsers and local"))
	fmt.Fprintln(&b, ui.StepHint("system so you can use HTTPS locally."))
	return b.String()
}

type ProvisionService struct {
	Name, ServerType string

	Domains []string

	// TODO(wes): ShowHints field

	finished bool

	spinner spinner.Model
}

func (m *ProvisionService) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type ServiceProvisionedMsg struct{}

func (m *ProvisionService) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case ServiceProvisionedMsg:
		m.finished = true
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *ProvisionService) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepHint("Now we'll provision your application's resources on Anchor.dev and the HTTPS"))
	fmt.Fprintln(&b, ui.StepHint("certificates for your development environment."))

	if m.finished {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Created %s [%s] %s resources on Anchor.dev.",
			ui.Emphasize(m.Name),
			ui.Domains(m.Domains),
			m.ServerType)))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Creating %s [%s] %s resources on Anchor.dev… %s",
			ui.Emphasize(m.Name),
			ui.Domains(m.Domains),
			m.ServerType,
			m.spinner.View())))
	}
	return b.String()
}
