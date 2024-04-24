package models

import (
	"fmt"
	"io"
	"strings"

	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type TrustAuditHeader struct{}

func (m *TrustAuditHeader) Init() tea.Cmd { return nil }

func (m *TrustAuditHeader) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TrustAuditHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Audit CA Certificates in Your Local Trust Store(s) %s", ui.Whisper("`anchor trust audit`"))))
	return b.String()
}

type TrustAuditScanFinishedMsg bool

type TrustAuditScan struct {
	spinner spinner.Model

	finished bool
}

func (m *TrustAuditScan) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *TrustAuditScan) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TrustAuditScanFinishedMsg:
		m.finished = bool(msg)
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *TrustAuditScan) View() string {
	var b strings.Builder

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Scanning local and expected CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Scanned local and expected CA certificates."))

	return b.String()
}

type TrustAuditInfo struct {
	AuditInfo *truststore.AuditInfo
	Stores    []truststore.Store
}

func (m *TrustAuditInfo) Init() tea.Cmd { return nil }

func (m *TrustAuditInfo) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TrustAuditInfo) View() string {
	var b strings.Builder

	for _, ca := range m.AuditInfo.Valid {
		fmt.Fprint(&b, ui.StepDone(fmt.Sprintf("%s - %s %s:",
			ui.Emphasize("VALID"),
			ui.Underline(ca.Subject.CommonName),
			ca.PublicKeyAlgorithm.String(),
		)))

		printStoresInfo(&b, m.AuditInfo, ca, m.Stores)

		fmt.Fprintln(&b)
	}

	for _, ca := range m.AuditInfo.Missing {
		fmt.Fprint(&b, ui.StepDone(fmt.Sprintf("%s - %s %s:",
			ui.Emphasize("MISSING"),
			ui.Underline(ca.Subject.CommonName),
			ca.PublicKeyAlgorithm.String(),
		)))

		printStoresInfo(&b, m.AuditInfo, ca, m.Stores)

		fmt.Fprintln(&b)
	}

	for _, ca := range m.AuditInfo.Extra {
		fmt.Fprint(&b, ui.StepDone(fmt.Sprintf("%s - %s %s:",
			ui.Emphasize("EXTRA"),
			ui.Underline(ca.Subject.CommonName),
			ca.PublicKeyAlgorithm.String(),
		)))

		printStoresInfo(&b, m.AuditInfo, ca, m.Stores)

		fmt.Fprintln(&b)
	}

	return b.String()
}

func printStoresInfo(w io.Writer, auditInfo *truststore.AuditInfo, ca *truststore.CA, stores []truststore.Store) {
	var missingStores, trustedStores []string
	for _, store := range stores {
		if auditInfo.IsPresent(ca, store) {
			trustedStores = append(trustedStores, store.Description())
		} else {
			missingStores = append(missingStores, store.Description())
		}
	}
	if len(missingStores) > 0 {
		fmt.Fprintf(w, " missing from [%s]", ui.Whisper(strings.Join(missingStores, ", ")))
	}
	if len(trustedStores) > 0 {
		fmt.Fprintf(w, " trusted by [%s]", ui.Whisper(strings.Join(trustedStores, ", ")))
	}
}
