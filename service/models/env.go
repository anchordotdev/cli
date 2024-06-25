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
			ui.StepHint("Environment variables provide your configuration and credentials."),
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

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Fetched %s environment variables.", ui.Emphasize(m.Service))))

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
			Key:    "clipboard",
			String: "Add to your clipboard",
		},
		{
			Key:    "dotenv",
			String: "Write to .env file",
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

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s environment variable management.", ui.Emphasize(m.choice))))
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
		fmt.Fprintln(&b, ui.StepAlert("Unable to copy env to your clipboard."))
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(
		"Copied %s env to your clipboard.",
		ui.Emphasize(m.Service))))

	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to load env in current session.",
		ui.Action("Paste and press enter"))))

	return b.String()
}

type EnvDotenv struct {
	Service string
}

func (m *EnvDotenv) Init() tea.Cmd { return nil }

func (m *EnvDotenv) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvDotenv) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf(
		"Wrote %s env to %s.",
		ui.Emphasize(m.Service),
		ui.Whisper("`.env`"))))

	return b.String()
}

type EnvNextSteps struct {
	LclUrl string
}

func (m *EnvNextSteps) Init() tea.Cmd { return nil }

func (m *EnvNextSteps) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *EnvNextSteps) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Header("Next Steps"))
	fmt.Fprintln(&b, ui.StepNext(fmt.Sprintf("(Re)Start your server and check out your encrypted site at: %s", ui.URL(m.LclUrl))))
	fmt.Fprintln(&b, ui.StepNext("These certificates will renew automatically, time to enjoy effortless encryption."))

	return b.String()
}
