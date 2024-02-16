package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SyncPreflight struct {
	NonInteractive bool
	ConfirmCh      chan<- struct{}

	step preflightStep

	expectedCAs []*truststore.CA
	auditInfo   *truststore.AuditInfo

	spinner spinner.Model
}

func (m *SyncPreflight) Init() tea.Cmd {
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

type (
	AuditInfoMsg *truststore.AuditInfo

	PreflightFinishedMsg struct{}
)

func (m *SyncPreflight) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AuditInfoMsg:
		m.auditInfo, m.step = msg, confirmSync

		if len(m.auditInfo.Missing) == 0 {
			m.step = noSync
		}

		return m, nil
	case PreflightFinishedMsg:
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

func (m *SyncPreflight) View() string {
	var b strings.Builder

	if m.step == diff {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Comparing local stores to expected CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	switch m.step {
	case confirmSync:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Compared local stores to expected CA certificates: need to install %d missing certificates.", len(m.auditInfo.Missing))))

		if m.NonInteractive {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Installing %d missing certificates. (%s)", len(m.auditInfo.Missing), ui.Accentuate("requires sudo"))))
		} else if m.ConfirmCh != nil {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to install %d missing certificates. (%s)", ui.Action("Press Enter"), len(m.auditInfo.Missing), ui.Accentuate("requires sudo"))))
		}
	case noSync:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Compared local stores to expected CA certificates: no changes needed.")))
	default:
		panic("impossible")
	}
	return b.String()
}

type SyncInstallCA struct {
	CA *truststore.CA

	stores    []truststore.Store
	installed map[truststore.Store]struct{}

	spinner spinner.Model
}

func (m *SyncInstallCA) Init() tea.Cmd {
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

type (
	SyncInstallingCAMsg struct {
		truststore.Store
	}

	SyncInstalledCAMsg struct {
		truststore.Store
	}
)

func (m *SyncInstallCA) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SyncInstalledCAMsg:
		if m.installed == nil {
			m.installed = map[truststore.Store]struct{}{}
		}
		m.installed[msg.Store] = struct{}{}
		return m, nil
	case SyncInstallingCAMsg:
		m.stores = append(m.stores, msg.Store)
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SyncInstallCA) View() string {
	commonName := m.CA.Subject.CommonName
	algo := m.CA.PublicKeyAlgorithm

	var b strings.Builder

	var installed []string
	var installing []string

	for _, store := range m.stores {
		if _, ok := m.installed[store]; ok {
			installed = append(installed, ui.Emphasize(store.Description()))
		} else {
			installing = append(installing, ui.Emphasize(store.Description()))
		}
	}

	if len(installed) > 0 {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Installed \"%s\" %s in %s.",
			ui.Underline(commonName),
			algo,
			strings.Join(installed, ", "),
		)))
	}
	if len(installing) > 0 {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Installing \"%s\" %s in %s.",
			ui.Underline(commonName),
			algo,
			strings.Join(installing, ", "),
		)))
	}

	return b.String()
}
