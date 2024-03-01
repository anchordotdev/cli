package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type TrustSignInHint struct{}

func (TrustSignInHint) Init() tea.Cmd { return nil }

func (m *TrustSignInHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *TrustSignInHint) View() string {
	var b strings.Builder
	// FIXME: first line duplicated from SignInHint, should dedup somehow
	fmt.Fprintln(&b, ui.StepHint("Please sign up or sign in with your Anchor account."))
	fmt.Fprintln(&b, ui.StepHint(""))
	fmt.Fprintln(&b, ui.StepHint("Once authenticated, we can lookup the CAs to trust."))
	return b.String()
}

type preflightStep int

const (
	diff preflightStep = iota
	finishedPreflight

	confirmSync
	noSync
)

type (
	HandleMsg string

	ExpectedCAsMsg []*truststore.CA
	TargetCAsMsg   []*truststore.CA
)

type TrustPreflight struct {
	Config *cli.Config

	ConfirmCh chan<- struct{}

	step preflightStep

	expectedCAs []*truststore.CA
	auditInfo   *truststore.AuditInfo

	spinner spinner.Model
}

func (m *TrustPreflight) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type (
	AuditInfoMsg *truststore.AuditInfo

	PreflightFinishedMsg struct{}
)

func (m *TrustPreflight) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m *TrustPreflight) View() string {
	var b strings.Builder

	if m.step == diff {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Comparing local and expected CA certificates…%s", m.spinner.View())))
		return b.String()
	}

	switch m.step {
	case confirmSync:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Compared local and expected CA certificates: need to install %d missing certificates.", len(m.auditInfo.Missing))))

		if m.Config.NonInteractive {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Installing %d missing certificates. (%s)", len(m.auditInfo.Missing), ui.Accentuate("requires sudo"))))
		} else if m.ConfirmCh != nil {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to install %d missing certificates. (%s)", ui.Action("Press Enter"), len(m.auditInfo.Missing), ui.Accentuate("requires sudo"))))
		}
	case noSync:
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Compared local and expected CA certificates: found matching certificates.")))
	default:
		panic("impossible")
	}
	return b.String()
}

type TrustUpdateStore struct {
	Store truststore.Store

	installing *truststore.CA
	installed  map[string][]string

	spinner spinner.Model
}

func (m *TrustUpdateStore) Init() tea.Cmd {
	m.installed = make(map[string][]string)
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

type (
	TrustStoreInstallingCAMsg struct {
		truststore.CA
	}

	TrustStoreInstalledCAMsg struct {
		truststore.CA
	}
)

func (m *TrustUpdateStore) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TrustStoreInstallingCAMsg:
		m.installing = &msg.CA
		return m, nil
	case TrustStoreInstalledCAMsg:
		m.installing = nil
		m.installed[msg.CA.Subject.CommonName] = append(m.installed[msg.CA.Subject.CommonName], msg.CA.PublicKeyAlgorithm.String())
		return m, nil
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *TrustUpdateStore) View() string {
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
