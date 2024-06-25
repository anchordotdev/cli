package models

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	LclConfigSkip = ui.Section{
		Name: "LclConfigSkip",
		Model: ui.MessageLines{
			ui.Skip("Configure System for lcl.host Local Development `anchor lcl config`"),
		},
	}

	LclConfigHeader = ui.Section{
		Name: "LclConfigHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Configure System for lcl.host HTTPS Local Development %s", ui.Whisper("`anchor lcl config`"))),
		},
	}

	LclConfigHint = ui.Section{
		Name: "LclConfigHint",
		Model: ui.MessageLines{
			ui.StepHint("Before issuing HTTPS certificates, we need to configure your browsers"),
			ui.StepHint("and OS to trust your personal certificates."),
			ui.Whisper("    |"), // whisper instead of stephint to avoid whitespace errors from git + golden
			ui.StepHint("We'll start a local diagnostic web server to guide you through the process."),
		},
	}

	Browserless = ui.Section{
		Name: "Browserless",
		Model: ui.MessageLines{
			ui.Warning("Unable to open browser, skipping browser-based verification."),
		},
	}
)

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

		fmt.Fprintln(&b, ui.StepHint("Next, we'll add your personal CA certificates to your system's trust stores."))

		return b.String()
	}

	return b.String()
}

type LclConfigSuccess struct {
	URL *url.URL
}

func (LclConfigSuccess) Init() tea.Cmd { return nil }

func (m LclConfigSuccess) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m LclConfigSuccess) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Success! %s works as expected (%s).",
		ui.URL(m.URL.String()),
		ui.Accentuate("encrypted with HTTPS"),
	)))

	return b.String()
}
