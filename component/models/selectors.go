package models

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anchordotdev/cli/ui"
)

type Choosable interface {
	comparable

	Key() string
	String() string

	Singular() string
	Plural() string
}

type Fetcher[T Choosable] struct {
	Flag, Plural, Singular string

	Creatable bool

	items []T

	spinner spinner.Model
}

func (m *Fetcher[T]) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *Fetcher[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []T:
		m.items = msg
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *Fetcher[T]) View() string {
	if m.items == nil {
		return fmt.Sprintln(
			ui.StepInProgress(fmt.Sprintf("Fetching %s…%s",
				m.Plural,
				m.spinner.View(),
			)),
		)
	}

	switch len(m.items) {
	case 0:
		if m.Creatable {
			return fmt.Sprintln(
				ui.StepDone(fmt.Sprintf("No %s found, so we'll create one.",
					m.Plural,
				)),
			)
		}

		return fmt.Sprintln(
			ui.StepAlert(fmt.Sprintf("No %s found!",
				m.Plural,
			)),
		)
	case 1:
		item := m.items[0]

		if !m.Creatable {
			return fmt.Sprintln(
				ui.StepDone(fmt.Sprintf("Using %s, the only available %s. %s",
					ui.Emphasize(item.Key()),
					m.Singular,
					ui.Whisper(
						fmt.Sprintf("You can also use `%s %s`.",
							m.Flag,
							item.Key(),
						),
					),
				)),
			)
		}
	}
	return ""
}

type Selector[T Choosable] struct {
	Prompt  string
	Flag    string
	Choices []ui.ListItem[T]

	ChoiceCh chan<- T

	chosen ui.ListItem[T]
	list   list.Model
}

func (m *Selector[T]) Init() tea.Cmd {
	m.list = ui.List(m.Choices)

	return nil
}

func (m *Selector[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if item, ok := m.list.SelectedItem().(ui.ListItem[T]); ok {
				if m.ChoiceCh != nil {
					m.chosen = item
					m.ChoiceCh <- m.chosen.Value
					close(m.ChoiceCh)
					m.ChoiceCh = nil
				}
			}
		case tea.KeyEsc:
			return m, ui.Exit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Selector[T]) View() string {
	var b strings.Builder

	if m.ChoiceCh != nil {
		fmt.Fprintln(&b, ui.StepPrompt(m.Prompt))
		fmt.Fprintln(&b, m.list.View())
		return b.String()
	}

	if reflect.ValueOf(m.chosen.Value).IsZero() {
		fmt.Fprintln(&b, ui.StepDone(
			fmt.Sprintf("Selected %s.", ui.Emphasize(m.chosen.String)),
		))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(
		fmt.Sprintf("Selected %s %s. %s",
			ui.Emphasize(m.chosen.Key),
			m.chosen.Value.Singular(),
			ui.Whisper(fmt.Sprintf("You can also use `%s %s`.", m.Flag, m.chosen.Key)),
		),
	))

	return b.String()
}
