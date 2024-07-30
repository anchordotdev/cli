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

var (
	TrustHeader = ui.Section{
		Name: "TrustHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Manage CA Certificates in your Local Trust Store(s) %s", ui.Whisper("`anchor trust`"))),
		},
	}

	TrustHint = ui.Section{
		Name: "TrustHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll check your local trust stores and make any needed updates."),
		},
	}
)

type TrustUpdateConfirm struct {
	ConfirmCh chan<- struct{}
}

func (m *TrustUpdateConfirm) Init() tea.Cmd { return nil }

func (m *TrustUpdateConfirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.ConfirmCh != nil {
				close(m.ConfirmCh)
				m.ConfirmCh = nil
			}
		}
	}

	return m, nil
}

func (m *TrustUpdateConfirm) View() string {
	var b strings.Builder

	if m.ConfirmCh != nil {
		fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("%s %s",
			ui.Accentuate("Updates may require sudo privileges, learn why here:"),
			ui.URL("https://lcl.host/why-sudo"),
		)))
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to install missing certificates. (%s)", ui.Action("Press Enter"), ui.Accentuate("requires sudo"))))

		return b.String()
	}

	return b.String()
}

type TrustUpdateStore struct {
	Config *cli.Config

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

	if m.installing != nil {
		// present thumbprint for comparison with pop up prompt
		if m.Config.GOOS() == "windows" {
			fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("\"%s\" Thumbprint (sha1): %s",
				m.installing.Subject.CommonName,
				m.installing.WindowsThumbprint(),
			)))
		}

		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Updating %s: installing %s %s… %s",
			ui.Emphasize(m.Store.Description()),
			ui.Underline(m.installing.Subject.CommonName),
			ui.Whisper(m.installing.PublicKeyAlgorithm.String()),
			m.spinner.View())))
	}

	return b.String()
}

type VMHint struct{}

func (m *VMHint) Init() tea.Cmd { return nil }

func (m *VMHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *VMHint) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepWarning("Running trust inside a VM or container will not update the host system."))
	fmt.Fprintln(&b, ui.StepHint("Rerun this command on your host system to update it's trust stores and enable")) // enable secure communication."))
	fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("secure communication, learn more here: %s", ui.URL("https://cl.host/vm-container-setup"))))
	return b.String()
}
