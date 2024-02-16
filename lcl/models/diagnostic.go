package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type Diagnostic struct {
	ConfirmCh chan<- struct{}

	Domain, Port, Scheme string
	ShowHeader           bool

	confirmCh chan<- struct{}
	url       string
}

func (Diagnostic) Init() tea.Cmd { return nil }

type OpenURLMsg string

func (m *Diagnostic) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m Diagnostic) View() string {
	var b strings.Builder

	if m.url == "" {
		return b.String()
	}

	var schemeMessage string
	if m.Scheme == "http" {
		schemeMessage = "without HTTPS"
	} else {
		schemeMessage = "with HTTPS"
	}

	if m.confirmCh != nil {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to test %s. (%s)",
			ui.Action("Press Enter"),
			ui.URL(m.url),
			ui.Accentuate(schemeMessage))))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Tested %s. (%s)",
			ui.URL(m.url),
			ui.Accentuate(schemeMessage),
		)))
	}

	return b.String()
}

type DiagnosticSuccess struct {
	Org, Realm, CA string
}

func (DiagnosticSuccess) Init() tea.Cmd { return nil }

func (m DiagnosticSuccess) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m DiagnosticSuccess) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepDone("Confirmed your lcl.host Local HTTPS setup."))
	return b.String()
}
