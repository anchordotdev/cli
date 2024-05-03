package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
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
	fmt.Fprintln(&b, ui.StepHint("configure your browsers and OS to trust your personal certificates."))
	fmt.Fprintln(&b, ui.Whisper("    |")) // whisper instead of stephint to avoid whitespace errors from git + golden
	fmt.Fprintln(&b, ui.StepHint("We'll start a local diagnostic web server to guide you through the process."))
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

type Browserless struct{}

func (m *Browserless) Init() tea.Cmd { return nil }

func (m *Browserless) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *Browserless) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Warning("Unable to open browser, skipping browser-based verification."))
	return b.String()
}
