package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	LclSignInHint = ui.Section{
		Name: "LclSignInHint",
		Model: ui.MessageLines{
			ui.StepHint("Please sign up or sign in with your Anchor account."),
			ui.StepHint(""),
			ui.StepHint("Once authenticated, we can provision your personalized Anchor resources to"),
			ui.StepHint("power HTTPS in your local development environment."),
		},
	}

	LclPreamble = ui.Section{
		Name: "LclPreamble",
		Model: ui.MessageLines{
			ui.Hint("Let's set up fast and totally free lcl.host HTTPS!"),
		},
	}

	LclHeader = ui.Section{
		Name: "LclHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Setup lcl.host HTTPS Local Development Environment %s", ui.Whisper("`anchor lcl`"))),
		},
	}

	LclHint = ui.Section{
		Name: "LclHint",
		Model: ui.MessageLines{
			ui.StepHint("After setup, you can use HTTPS locally in your browsers and other programs."),
		},
	}
)

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

	fmt.Fprintln(&b, ui.StepHint("Now we'll provision Anchor.dev resources and HTTPS certificates for you."))

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

type DomainInput struct {
	InputCh chan<- string

	Default, Domain, TLD string
	Prompt, Done         string

	input *textinput.Model
}

func (m *DomainInput) Init() tea.Cmd {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.Style = ui.Prompt
	ti.Focus()
	ti.KeyMap.AcceptSuggestion = key.Binding{}
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

				if isValidDomain(value) {
					if strings.HasSuffix(value, "."+m.TLD) {
						m.Domain = value
					} else {
						m.Domain = value + "." + m.TLD
					}
					m.InputCh <- m.Domain
					m.InputCh = nil
				}
			}
			return m, nil
		case tea.KeyEsc:
			return m, ui.Exit
		default:
			if m.isValidDomainRunes(msg.Runes) {
				ti, cmd := m.input.Update(msg)
				m.input = &ti

				// if there is input and it doesn't already have the TLD, suggest it with the TLD
				if len(m.input.Value()) > 0 && !strings.HasSuffix(m.input.Value(), "."+m.TLD) {
					m.input.SetSuggestions([]string{m.input.Value() + "." + m.TLD})
				}

				return m, cmd
			}
		}
	}
	return m, nil
}

func isValidDomain(domain string) bool {
	if len(domain) == 0 {
		return false
	}

	firstLastChars := []byte{domain[0], domain[len(domain)-1]}
	if strings.ContainsAny(string(firstLastChars), "-.") {
		return false
	}

	return !strings.Contains(domain, "..")
}

var validDomainRunes = []rune{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'-', '.',
}

func (m *DomainInput) isValidDomainRunes(runes []rune) bool {
	for _, r := range runes {
		if !slices.Contains(validDomainRunes, r) {
			return false
		}
	}

	return true
}

func (m *DomainInput) View() string {
	var b strings.Builder

	if m.InputCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt(m.Prompt))
		fmt.Fprintln(&b, ui.StepHint("We will ignore any characters that are not valid in a domain."))
		fmt.Fprintln(&b, ui.StepPrompt(m.input.View()))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(m.Done, ui.Emphasize(m.Domain))))
	}

	return b.String()
}

type DomainResolver struct {
	Domain string

	finished, success bool

	spinner spinner.Model
}

func (m *DomainResolver) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type DomainStatusMsg bool

func (m *DomainResolver) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case DomainStatusMsg:
		m.finished = true
		m.success = bool(msg)
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *DomainResolver) View() string {
	var b strings.Builder

	switch {
	case !m.finished:
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Resolving %s domain…%s", ui.URL(m.Domain), m.spinner.View())))
	case m.success:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Resolved %s domain: success!", ui.URL(m.Domain))))
	default:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Resolved %s domain: failed!", ui.URL(m.Domain))))
		fmt.Fprintln(&b, ui.StepHint("The entered domain name is either invalid or cannot be resolved from"))
		fmt.Fprintln(&b, ui.StepHint("your machine, possibly due to rebinding protection on your DNS server."))
		fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("%s %s",
			ui.Accentuate("Learn more here:"),
			ui.URL("https://lcl.host/dns-rebinding"),
		)))
	}

	return b.String()
}
