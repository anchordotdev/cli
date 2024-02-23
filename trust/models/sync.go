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

type SyncUpdateStore struct {
	Store truststore.Store

	installing *truststore.CA
	installed  map[string][]string

	spinner spinner.Model
}

func (m *SyncUpdateStore) Init() tea.Cmd {
	m.installed = make(map[string][]string)
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

type (
	SyncStoreInstallingCAMsg struct {
		truststore.CA
	}

	SyncStoreInstalledCAMsg struct {
		truststore.CA
	}
)

func (m *SyncUpdateStore) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SyncStoreInstallingCAMsg:
		m.installing = &msg.CA
		return m, nil
	case SyncStoreInstalledCAMsg:
		m.installing = nil
		m.installed[msg.CA.Subject.CommonName] = append(m.installed[msg.CA.Subject.CommonName], msg.CA.PublicKeyAlgorithm.String())
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SyncUpdateStore) View() string {
	var b strings.Builder

	if m.installing != nil {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Updating %s: installing %s %s.",
			ui.Emphasize(m.Store.Description()),
			ui.Underline(m.installing.Subject.CommonName),
			ui.Whisper(m.installing.PublicKeyAlgorithm.String()),
		)))
	}
	if len(m.installed) > 0 {
		var styledCAs []string

		for subjectCommonName, algorithms := range m.installed {
			styledCAs = append(styledCAs, fmt.Sprintf("%s [%s]",
				ui.Underline(subjectCommonName),
				ui.Whisper(strings.Join(algorithms, ", ")),
			))
		}

		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Updated %s: installed %s",
			ui.Emphasize(m.Store.Description()),
			strings.Join(styledCAs, ", "),
		)))
	}

	return b.String()
}
