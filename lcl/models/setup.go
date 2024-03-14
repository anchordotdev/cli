package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anchordotdev/cli/detection"
	"github.com/anchordotdev/cli/ui"
)

type SetupHeader struct{}

func (m *SetupHeader) Init() tea.Cmd { return nil }

func (m *SetupHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SetupHeader) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Setup lcl.host Application %s", ui.Whisper("`anchor lcl setup`"))))

	return b.String()
}

type SetupHint struct{}

func (m *SetupHint) Init() tea.Cmd { return nil }

func (m *SetupHint) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SetupHint) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepHint("We'll start by scanning your current directory, then ask you questions about"))
	fmt.Fprintln(&b, ui.StepHint("your local application so that we can generate setup instructions for you."))

	return b.String()
}

type SetupScan struct {
	finished bool
	spinner  spinner.Model
}

func (m *SetupScan) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *SetupScan) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case detection.Results:
		m.finished = true

		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *SetupScan) View() string {
	var b strings.Builder

	if !m.finished {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Scanning current directory for local application…%s", m.spinner.View())))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone("Scanned current directory."))

	return b.String()
}

type SetupCategory struct {
	ChoiceCh chan<- string
	Results  detection.Results

	list   list.Model
	choice string
}

func (m *SetupCategory) Init() tea.Cmd {
	var items []ui.ListItem[string]
	for _, match := range m.Results[detection.High] {
		item := ui.ListItem[string]{
			Key:   match.AnchorCategory.Key,
			Value: ui.Titlize(match.Detector.GetTitle()),
		}

		items = append(items, item)
	}

	for _, match := range m.Results[detection.Medium] {
		item := ui.ListItem[string]{
			Key:   match.AnchorCategory.Key,
			Value: match.Detector.GetTitle(),
		}

		items = append(items, item)
	}
	for _, match := range m.Results[detection.Low] {
		item := ui.ListItem[string]{
			Key:   match.AnchorCategory.Key,
			Value: ui.Whisper(match.Detector.GetTitle()),
		}

		items = append(items, item)
	}

	for _, match := range m.Results[detection.None] {
		item := ui.ListItem[string]{
			Key:   match.AnchorCategory.Key,
			Value: ui.Whisper(match.Detector.GetTitle()),
		}

		items = append(items, item)
	}

	m.list = ui.List(items)

	return nil
}

func (m *SetupCategory) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)

			if item, ok := m.list.SelectedItem().(ui.ListItem[string]); ok {
				m.choice = item.Key
				if m.ChoiceCh != nil {
					m.ChoiceCh <- m.choice
					close(m.ChoiceCh)
					m.ChoiceCh = nil
				}
			}

			return m, cmd
		case tea.KeyEsc:
			return m, ui.Exit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *SetupCategory) View() string {
	var b strings.Builder

	if m.ChoiceCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("What application server type?"))
		fmt.Fprintln(&b, m.list.View())

		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s application server type", ui.Emphasize(m.choice))))

	return b.String()
}

type SetupName struct {
	InputCh chan<- string

	Default string

	input  *textinput.Model
	choice string
}

func (m *SetupName) Init() tea.Cmd {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.Style = ui.Prompt
	ti.Focus()

	if len(m.Default) > 0 {
		ti.Placeholder = m.Default
	}

	m.input = &ti

	return textinput.Blink
}

func (m *SetupName) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.InputCh != nil {
				value := m.input.Value()
				if value == "" {
					value = m.Default
				}

				m.choice = value
				m.InputCh <- value
				m.InputCh = nil
			}
			return m, nil
		case tea.KeyEsc:
			return m, ui.Exit
		}
	}

	ti, cmd := m.input.Update(msg)
	m.input = &ti
	return m, cmd
}

func (m *SetupName) View() string {
	var b strings.Builder

	if m.InputCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("What is the application name?"))
		fmt.Fprintln(&b, ui.StepPrompt(m.input.View()))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s application name.", ui.Emphasize(m.choice))))

	return b.String()
}

type SetupGuidePrompt struct {
	ConfirmCh chan<- struct{}

	confirmCh chan<- struct{}
	url       string
}

type OpenSetupGuideMsg string

func (SetupGuidePrompt) Init() tea.Cmd { return nil }

func (m *SetupGuidePrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case OpenSetupGuideMsg:
		if m.url == "" {
			m.url = string(msg)
			m.confirmCh = m.ConfirmCh
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.confirmCh != nil {
				close(m.confirmCh)
				m.confirmCh = nil
			}
		}
	}

	return m, nil
}

func (m SetupGuidePrompt) View() string {
	var b strings.Builder

	if m.url == "" {
		return b.String()
	}

	fmt.Fprintln(&b, ui.Header("Next Steps"))
	fmt.Fprintln(&b, ui.StepHint("Now that you have local HTTPS setup, let's automate certificate provisioning"))
	fmt.Fprintln(&b, ui.StepHint("so you never have to manually provision future certificates again."))
	fmt.Fprintln(&b, ui.StepHint(""))
	fmt.Fprintln(&b, ui.StepHint("We've generated an Anchor.dev setup guide for your application with"))
	fmt.Fprintln(&b, ui.StepHint("instructions for automating certificate provisioning inside your application."))

	if m.confirmCh != nil {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to open %s.",
			ui.Action("Press Enter"),
			ui.URL(m.url),
		)))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Opened %s.", ui.URL(m.url))))
	}

	return b.String()
}
