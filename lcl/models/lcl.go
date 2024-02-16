package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LclPreamble struct{}

func (LclPreamble) Init() tea.Cmd { return nil }

func (m LclPreamble) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclPreamble) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint("Anchor + lcl.host provides fast, free, magic HTTPS for local applications and services."))
	fmt.Fprintln(&b, ui.Hint(""))
	fmt.Fprintln(&b, ui.Hint("Let's setup your HTTPS secured development environment and dev/prod parity!"))
	return b.String()
}

type (
	ScanFinishedMsg struct{}
)

type LclScan struct {
	finished bool

	spinner spinner.Model
}

func (m *LclScan) Init() tea.Cmd {
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

func (m *LclScan) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ScanFinishedMsg:
		m.finished = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *LclScan) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header("Set Up lcl.host Local HTTPS Diagnostic"))
	fmt.Fprintln(&b, ui.StepHint("We will start by determining your system's starting point for setup."))

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Scanning local configuration and status…%s", m.spinner.View())))
	} else {
		fmt.Fprintln(&b, ui.StepDone("Scanned local configuration and status."))
	}
	return b.String()
}

type DomainInput struct {
	InputCh chan<- string

	Default    string
	Domain     string
	TLD        string
	SkipHeader bool

	input *textinput.Model
}

func (m *DomainInput) Init() tea.Cmd {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.Style = ui.Prompt
	ti.Focus()
	ti.ShowSuggestions = true

	if len(m.Default) > 0 {
		ti.Placeholder = m.Default + "." + m.TLD
	}

	m.input = &ti

	return textinput.Blink
}

func (m *DomainInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.InputCh != nil {
				value := m.input.Value()
				if value == "" {
					value = m.Default
				}

				m.Domain = value
				m.InputCh <- value
				m.InputCh = nil
			}
			return m, nil
		case tea.KeyCtrlC, tea.KeyEsc:
			// TODO: double-check if this is necessary
			return m, tea.Quit
		}
	}

	if len(m.input.Value()) > 0 {
		m.input.SetSuggestions([]string{m.input.Value() + "." + m.TLD})
	}

	ti, cmd := m.input.Update(msg)
	m.input = &ti
	return m, cmd
}

func (m *DomainInput) View() string {
	var b strings.Builder

	if m.InputCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("What lcl.host domain would you like to use for diagnostics?"))
		fmt.Fprintln(&b, ui.StepPrompt(m.input.View()))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s domain for lcl.host diagnostic certificate.", ui.Emphasize(m.Domain+".lcl.host"))))
	}

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
	m.spinner = ui.Spinner()

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
	if m.finished {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Created %s [%s] %s server resources on Anchor.dev.",
			ui.Emphasize(m.Name),
			ui.Domains(m.Domains),
			m.ServerType)))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Creating %s [%s] %s server resources on Anchor.dev… %s",
			ui.Emphasize(m.Name),
			ui.Domains(m.Domains),
			m.ServerType,
			m.spinner.View())))
	}
	return b.String()
}
