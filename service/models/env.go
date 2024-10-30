package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	ServiceEnvHeader = ui.Section{
		Name: "ServiceEnvHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Fetch Environment Variables for Service %s", ui.Whisper("`anchor service env`"))),
		},
	}

	ServiceEnvHint = ui.Section{
		Name: "ServiceEnvHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll set your environment variables to provide configuration and credentials."),
		},
	}
)

type EnvFetchedMsg struct{}

type EnvFetch struct {
	finished bool
	Service  string

	spinner spinner.Model
}

func (m *EnvFetch) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *EnvFetch) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case EnvFetchedMsg:
		m.finished = true
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *EnvFetch) View() string {
	var b strings.Builder

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Fetching %s environment variables…%s",
			ui.Emphasize(m.Service),
			m.spinner.View())))
		return b.String()
	}

	return b.String()
}

type EnvMethod struct {
	ChoiceCh chan<- string

	choice string
	list   list.Model
}

func (m *EnvMethod) Init() tea.Cmd {
	m.list = ui.List([]ui.ListItem[string]{
		{
			Key:    "export",
			String: "Add export commands to your clipboard.",
		},
		{
			Key:    "dotenv",
			String: "Add dotenv contents to your clipboard.",
		},
		{
			Key:    "display",
			String: "Display export commands. ! WARNING: could be observed by others.",
		},
	})
	return nil
}

func (m *EnvMethod) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if item, ok := m.list.SelectedItem().(ui.ListItem[string]); ok {
				m.choice = item.Key
				if m.ChoiceCh != nil {
					m.ChoiceCh <- m.choice
					close(m.ChoiceCh)
					m.ChoiceCh = nil
				}
			}
		case tea.KeyEscape:
			return m, ui.Exit
		}
	}

	return m, cmd
}

func (m *EnvMethod) View() string {
	var b strings.Builder

	if m.ChoiceCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("How would you like to manage your environment variables?"))
		fmt.Fprintln(&b, m.list.View())
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(
		fmt.Sprintf("Selected %s environment variable method. %s",
			ui.Emphasize(m.choice),
			ui.Whisper(fmt.Sprintf("You can also use `--method %s`.", m.choice)),
		),
	))
	return b.String()
}

type EnvClipboard struct {
	InClipboard bool
	Service     string
}

func (m *EnvClipboard) Init() tea.Cmd { return nil }

func (m *EnvClipboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvClipboard) View() string {
	var b strings.Builder

	if !m.InClipboard {
		// FIXME: handling for clipboard errors
		fmt.Fprintln(&b, ui.StepAlert("Unable to copy export commands to your clipboard."))
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(
		"Copied %s export commands to your clipboard.",
		ui.Emphasize(m.Service))))

	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to load your environment variables.",
		ui.Action("Paste and press enter"))))

	return b.String()
}

type EnvDotenv struct {
	InClipboard bool
	Service     string
}

func (m *EnvDotenv) Init() tea.Cmd { return nil }

func (m *EnvDotenv) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvDotenv) View() string {
	var b strings.Builder

	if !m.InClipboard {
		// FIXME: handling for clipboard errors
		fmt.Fprintln(&b, ui.StepAlert("Unable to copy dotenv contents to your clipboard."))
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(
		"Copied %s dotenv contents to your clipboard.",
		ui.Emphasize(m.Service))))

	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s into your dotenv so your server will find them next restart.",
		ui.Action("Paste"))))

	return b.String()
}

type EnvDisplay struct {
	EnvString string
	Service   string
}

func (m *EnvDisplay) Init() tea.Cmd { return nil }

func (m *EnvDisplay) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvDisplay) View() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n%s\n", m.EnvString)

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(
		"Displayed %s export commands.",
		ui.Emphasize(m.Service))))

	fmt.Fprintln(&b, ui.StepAlert("Be sure to load these into your environment."))

	return b.String()
}

type EnvNextSteps struct {
	LclUrl, OrgApid, RealmApid, ServiceApid string
}

func (m *EnvNextSteps) Init() tea.Cmd { return nil }

func (m *EnvNextSteps) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvNextSteps) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Header("Next Steps"))
	fmt.Fprintln(&b, ui.StepAlert(ui.Action("(Re)Start your server.")))
	if m.LclUrl != "" {
		fmt.Fprintln(&b, ui.StepAlert(
			fmt.Sprintf("%s: run `anchor service verify --org %s --realm %s --service %s`.",
				ui.Action("Verify TLS setup and configuration"),
				m.OrgApid,
				m.RealmApid,
				m.ServiceApid,
			)))
	}
	fmt.Fprintln(&b, ui.StepHint("These certificates will renew automatically, time to enjoy effortless encryption."))

	return b.String()
}
