package models

import (
	"fmt"
	"slices"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
)

type OrgCreateHeader struct {
	hidden bool
}

func (m *OrgCreateHeader) Init() tea.Cmd { return nil }

func (m *OrgCreateHeader) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.HideModelsMsg:
		if slices.Contains(msg.Models, "OrgCreateHeader") {
			m.hidden = true
		}
	}

	return m, nil
}

func (m *OrgCreateHeader) View() string {
	if m.hidden {
		return ""
	}

	var b strings.Builder

	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Create New Organization %s", ui.Whisper("`anchor org create`"))))

	return b.String()
}

type OrgCreateHint struct {
	hidden bool
}

func (m *OrgCreateHint) Init() tea.Cmd { return nil }

func (m *OrgCreateHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.HideModelsMsg:
		if slices.Contains(msg.Models, "OrgCreateHint") {
			m.hidden = true
		}
	}

	return m, nil
}

func (m *OrgCreateHint) View() string {
	if m.hidden {
		return ""
	}

	var b strings.Builder

	fmt.Fprintln(&b, ui.StepHint("We'll create a new organization to facilitate collaboration."))

	return b.String()
}

type CreateOrgNameInput struct {
	InputCh chan<- string

	hidden bool
	input  *textinput.Model
	choice string
}

func (m *CreateOrgNameInput) Init() tea.Cmd {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Cursor.Style = ui.Prompt
	ti.Focus()

	m.input = &ti

	return textinput.Blink
}

func (m *CreateOrgNameInput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.HideModelsMsg:
		if slices.Contains(msg.Models, "CreateOrgNameInput") {
			m.hidden = true
		}
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.InputCh != nil {
				value := m.input.Value()

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

func (m *CreateOrgNameInput) View() string {
	if m.hidden {
		return ""
	}

	var b strings.Builder

	if m.InputCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt("What is the new organization's name?"))
		fmt.Fprintln(&b, ui.StepPrompt(m.input.View()))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Entered %s organization name.", ui.Emphasize(m.choice))))

	return b.String()
}

type CreateOrgSpinner struct {
	hidden  bool
	spinner spinner.Model
}

func (m *CreateOrgSpinner) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *CreateOrgSpinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ui.HideModelsMsg:
		if slices.Contains(msg.Models, "CreateOrgSpinner") {
			m.hidden = true
		}
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *CreateOrgSpinner) View() string {
	if m.hidden {
		return ""
	}

	var b strings.Builder

	fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Creating new organization…%s", m.spinner.View())))

	return b.String()
}

type CreateOrgResult struct {
	Org api.Organization
}

func (m *CreateOrgResult) Init() tea.Cmd { return nil }

func (m *CreateOrgResult) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *CreateOrgResult) View() string {
	var b strings.Builder

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Created %s %s organization.",
		m.Org.Name,
		ui.Whisper(fmt.Sprintf("(%s)", m.Org.Apid)),
	)))

	return b.String()
}
