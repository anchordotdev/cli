package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

type TrustCleanHeader struct{}

func (m *TrustCleanHeader) Init() tea.Cmd { return nil }

func (m *TrustCleanHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TrustCleanHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Clean CA Certificates from Local Trust Store(s) %s", ui.Whisper("`anchor trust clean`"))))
	return b.String()
}

type TrustCleanHint struct {
	CertStates, TrustStores []string

	spinner spinner.Model
}

func (c *TrustCleanHint) Init() tea.Cmd {
	c.spinner = ui.WaitingSpinner()

	return c.spinner.Tick
}

func (c *TrustCleanHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	c.spinner, cmd = c.spinner.Update(msg)
	return c, cmd
}

func (c *TrustCleanHint) View() string {
	states := strings.Join(c.CertStates, ", ")
	stores := strings.Join(c.TrustStores, ", ")

	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint(fmt.Sprintf("Removing %s CA certificates from the %s store(s).", states, stores)))

	return b.String()
}

type TrustCleanAudit struct {
	finished  bool
	spinner   spinner.Model
	targetCAs []*truststore.CA
}

func (m *TrustCleanAudit) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *TrustCleanAudit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []*truststore.CA:
		m.targetCAs = msg
		m.finished = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *TrustCleanAudit) View() string {
	var b strings.Builder

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Auditing local CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Audited local CA certificates: need to remove %d certificates.", len(m.targetCAs))))

	return b.String()
}

type TrustCleanCA struct {
	Config *cli.Config

	CA *truststore.CA

	ConfirmCh chan<- struct{}

	cleaning truststore.Store
	cleaned  []string

	spinner spinner.Model
}

func (m *TrustCleanCA) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type (
	CACleaningMsg struct {
		truststore.Store
	}
	CACleanedMsg struct {
		truststore.Store
	}
)

func (m *TrustCleanCA) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CACleaningMsg:
		m.cleaning = msg.Store
		return m, nil
	case CACleanedMsg:
		m.cleaning = nil
		m.cleaned = append(m.cleaned, msg.Store.Description())
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.ConfirmCh == nil {
				return m, nil
			}

			close(m.ConfirmCh)
			m.ConfirmCh = nil
		}
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *TrustCleanCA) View() string {
	var b strings.Builder

	if m.ConfirmCh != nil && !m.Config.NonInteractive {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to remove %s %s CA Certificate. (%s)",
			ui.Action("Press Enter"),
			ui.EmphasizeUnderline(m.CA.Subject.CommonName),
			ui.Emphasize(m.CA.PublicKeyAlgorithm.String()),
			ui.Accentuate("requires sudo"),
		)))
		return b.String()
	}

	if m.cleaning != nil {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Cleaning %s %s: removing from %s…%s",
			ui.EmphasizeUnderline(m.CA.Subject.CommonName),
			ui.Emphasize(m.CA.PublicKeyAlgorithm.String()),
			ui.Whisper(m.cleaning.Description()),
			m.spinner.View(),
		)))
		return b.String()
	}

	if len(m.cleaned) > 0 {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Cleaned %s %s: removed from %s",
			ui.EmphasizeUnderline(m.CA.Subject.CommonName),
			ui.Emphasize(m.CA.PublicKeyAlgorithm.String()),
			fmt.Sprintf("[%s]", ui.Whisper(strings.Join(m.cleaned, ", "))),
		)))
	}

	return b.String()
}
