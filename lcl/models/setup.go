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

var (
	SetupHeader = ui.Section{
		Name: "SetupHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Setup lcl.host Application %s", ui.Whisper("`anchor lcl setup`"))),
		},
	}

	SetupHint = ui.Section{
		Name: "SetupHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll integrate your application and system for HTTPS local development."),
		},
	}

	SetupAnchorTOML = ui.Section{
		Name: "AnchorTOML",
		Model: ui.MessageLines{
			ui.StepNext("Be sure to add anchor.toml to your version control system."),
		},
	}
)

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
			Key:    match.AnchorCategory.Key,
			String: ui.Titlize(match.Detector.GetTitle()),
		}

		items = append(items, item)
	}

	for _, match := range m.Results[detection.Medium] {
		item := ui.ListItem[string]{
			Key:    match.AnchorCategory.Key,
			String: match.Detector.GetTitle(),
		}

		items = append(items, item)
	}
	for _, match := range m.Results[detection.Low] {
		item := ui.ListItem[string]{
			Key:    match.AnchorCategory.Key,
			String: ui.Whisper(match.Detector.GetTitle()),
		}

		items = append(items, item)
	}

	for _, match := range m.Results[detection.None] {
		item := ui.ListItem[string]{
			Key:    match.AnchorCategory.Key,
			String: ui.Whisper(match.Detector.GetTitle()),
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

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s application server type.", ui.Emphasize(m.choice))))

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

type SetupMethod struct {
	ChoiceCh chan<- string

	list   list.Model
	choice string
}

func (m *SetupMethod) Init() tea.Cmd {
	m.list = ui.List([]ui.ListItem[string]{
		{
			Key:    "automated",
			String: "ACME Automated - Anchor style guides you through setup and automates renewal - Recommended",
		},
		{
			Key:    "manual",
			String: "Manual - mkcert style leaves setup and renewal up to you",
		},
	})
	return nil
}

func (m *SetupMethod) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case tea.KeyEsc:
			return m, ui.Exit
		}
	}

	return m, cmd
}

func (m *SetupMethod) View() string {
	var b strings.Builder

	if m.ChoiceCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("How would you like to manage your lcl.host certificates?"))
		fmt.Fprintln(&b, m.list.View())
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s certificate management.", ui.Emphasize(m.choice))))
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

	fmt.Fprintln(&b, ui.StepHint("Now follow your customized Anchor.dev setup guide to automate certificate"))
	fmt.Fprintln(&b, ui.StepHint("management so you'll never have to manually provision certificates again."))

	if m.confirmCh != nil {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to open %s in your browser.",
			ui.Action("Press Enter"),
			ui.URL(m.url),
		)))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Opened %s.", ui.URL(m.url))))
	}

	return b.String()
}

type SetupGuideHint struct {
	LclUrl string
}

func (m *SetupGuideHint) Init() tea.Cmd { return nil }

func (m *SetupGuideHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SetupGuideHint) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.Header("Next Steps"))
	fmt.Fprintln(&b, ui.StepNext(fmt.Sprintf("After following the guide, check out your encrypted site at: %s", ui.URL(m.LclUrl))))
	fmt.Fprintln(&b, ui.StepNext("These certificates will renew automatically, time to enjoy effortless encryption."))

	return b.String()
}
