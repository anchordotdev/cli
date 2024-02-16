package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Provision struct {
	Domains []string

	certFile, chainFile, keyFile string

	spinner spinner.Model
}

func (m *Provision) Init() tea.Cmd {
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

type ProvisionedFiles [3]string

func (m *Provision) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProvisionedFiles:
		m.certFile = msg[0]
		m.chainFile = msg[1]
		m.keyFile = msg[2]

		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *Provision) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header("Provision Certificate"))
	fmt.Fprintln(&b, ui.StepHint("You can manually use these certificate files or automate your certificates by following our setup guide."))

	if m.certFile == "" {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Provisioning certificate for [%s]… %s",
			ui.Domains(m.Domains), m.spinner.View())))

		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Provisioned certificate for [%s].", ui.Domains(m.Domains))))
	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Wrote certificate to %s", ui.Emphasize(m.certFile))))
	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Wrote chain to %s", ui.Emphasize(m.chainFile))))
	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Wrote key to %s", ui.Emphasize(m.chainFile))))

	return b.String()
}
