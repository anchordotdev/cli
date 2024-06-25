package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SelectorChoices[T comparable] interface {
	Flag() string
	Plural() string
	Singular() string

	ListItems() []ui.ListItem[T]
}

type SelectorFetcher[T comparable, U SelectorChoices[T]] struct {
	choices U

	spinner spinner.Model
}

func (m *SelectorFetcher[T, U]) Init() tea.Cmd {
	m.spinner = ui.WaitingSpinner()

	return m.spinner.Tick
}

func (m *SelectorFetcher[T, U]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case U:
		m.choices = msg
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SelectorFetcher[T, U]) View() string {
	var b strings.Builder

	// TODO: handle case where items is empty after fetch

	items := m.choices.ListItems()
	if len(items) == 0 {
		fmt.Fprintln(&b, ui.StepInProgress(
			fmt.Sprintf("Fetching %s…%s",
				m.choices.Plural(),
				m.spinner.View())))
		return b.String()
	}

	if len(items) == 1 {
		item := items[0]
		fmt.Fprintln(&b, ui.StepDone(
			fmt.Sprintf("Using %s, the only available %s. %s",
				ui.Emphasize(item.Key),
				m.choices.Singular(),
				ui.Whisper(fmt.Sprintf("You can also use `%s %s`.", m.choices.Flag(), item.Key)),
			)))
	}

	return b.String()
}

type Selector[T comparable] struct {
	ChoiceCh chan<- T
	Choices  SelectorChoices[T]
	Prompt   string

	choice ui.ListItem[T]
	list   list.Model
}

func (m *Selector[T]) Init() tea.Cmd {
	m.list = ui.List(m.Choices.ListItems())

	return nil
}

func (m *Selector[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if item, ok := m.list.SelectedItem().(ui.ListItem[T]); ok {
				m.choice = item
				if m.ChoiceCh != nil {
					m.ChoiceCh <- m.choice.Value
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

	fmt.Fprintln(&b, ui.StepDone(
		fmt.Sprintf("Selected %s %s. %s",
			ui.Emphasize(m.choice.Key),
			m.Choices.Singular(),
			ui.Whisper(fmt.Sprintf("You can also use `%s %s`.", m.Choices.Flag(), m.choice.Key)),
		)))

	return b.String()
}
