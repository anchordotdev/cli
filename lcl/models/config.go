package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LclConfigSkip struct{}

func (LclConfigSkip) Init() tea.Cmd { return nil }

func (m *LclConfigSkip) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *LclConfigSkip) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Skip("Configure System for lcl.host Local Development `anchor lcl config`"))
	return b.String()
}

type LclConfigHeader struct{}

func (LclConfigHeader) Init() tea.Cmd { return nil }

func (m LclConfigHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclConfigHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Configure System for lcl.host HTTPS Local Development %s", ui.Whisper("`anchor lcl config`"))))
	return b.String()
}

type LclConfigHint struct{}

func (LclConfigHint) Init() tea.Cmd { return nil }

func (m LclConfigHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclConfigHint) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepHint("Before issuing HTTPS certificates for your local applications, we need to"))
	fmt.Fprintln(&b, ui.StepHint("configure your browsers and OS to trust your personal certificates. "))
	fmt.Fprintln(&b, ui.StepHint(""))
	fmt.Fprintln(&b, ui.StepHint("We'll start a local diagnostic web server to guide you through the process."))
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
		case tea.KeyEsc:
			return m, ui.Exit
		default:
			if m.validDomainInput(msg.Runes) {
				ti, cmd := m.input.Update(msg)
				m.input = &ti

				if len(m.input.Value()) > 0 {
					m.input.SetSuggestions([]string{m.input.Value() + "." + m.TLD})
				}

				return m, cmd
			}
		}
	}
	return m, nil
}

var validDomainRunes = []rune{
	'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z',
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X', 'Y', 'Z',
	'0', '1', '2', '3', '4', '5', '6', '7', '8', '9',
	'-', '.',
}

func (m *DomainInput) validDomainInput(runes []rune) bool {
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
		fmt.Fprintln(&b, ui.StepPrompt("What lcl.host domain would you like to use for diagnostics?"))
		fmt.Fprintln(&b, ui.StepPrompt(m.input.View()))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s domain for lcl.host diagnostic certificate.", ui.Emphasize(m.Domain+".lcl.host"))))
	}

	return b.String()
}

type LclConfig struct {
	ConfirmCh chan<- struct{}

	Domain, Port, Scheme string
	ShowHeader           bool

	confirmCh chan<- struct{}
	url       string
}

func (LclConfig) Init() tea.Cmd { return nil }

type OpenURLMsg string

func (m *LclConfig) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m LclConfig) View() string {
	var b strings.Builder

	if m.url == "" {
		return b.String()
	}

	if m.Scheme == "https" {
		fmt.Fprintln(&b, ui.StepHint("Before we move on, let's test your system configuration by trying out HTTPS."))
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

		fmt.Fprintln(&b, ui.StepHint("Next, we'll add your personal CA certificates to your system's trust stores."))
		fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("%s %s",
			ui.Accentuate("This may require sudo privileges, learn why here: "),
			ui.URL("https://lcl.host/why-sudo"),
		)))

		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Success! %s works as expected (%s).",
		ui.URL(m.url),
		ui.Accentuate("encrypted with HTTPS"),
	)))

	return b.String()
}

type LclConfigSuccess struct {
	Org, Realm, CA string
}

func (LclConfigSuccess) Init() tea.Cmd { return nil }

func (m LclConfigSuccess) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclConfigSuccess) View() string {
	var b strings.Builder

	// TODO: move success part of Diagnostic to here

	return b.String()
}
