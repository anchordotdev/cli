package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type TrustStoreAudit struct {
	auditInfo *truststore.AuditInfo

	spinner spinner.Model
}

func (m *TrustStoreAudit) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type AuditInfoMsg *truststore.AuditInfo

func (m *TrustStoreAudit) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AuditInfoMsg:
		m.auditInfo = msg
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *TrustStoreAudit) View() string {
	var b strings.Builder

	if m.auditInfo == nil {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Comparing local and expected CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	if len(m.auditInfo.Missing) > 0 {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Compared local and expected CA certificates: need to install %d missing certificates.", len(m.auditInfo.Missing))))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Compared local and expected CA certificates: no updates needed."))
	return b.String()
}
