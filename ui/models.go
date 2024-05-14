package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type MessageLines []string

func (MessageLines) Init() tea.Cmd { return nil }

func (m MessageLines) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m MessageLines) View() string {
	var b strings.Builder
	for _, line := range m {
		fmt.Fprintln(&b, line)
	}
	return b.String()
}

type MessageFunc func(*strings.Builder)

func (MessageFunc) Init() tea.Cmd { return nil }

func (m MessageFunc) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m MessageFunc) View() string {
	var b strings.Builder
	m(&b)
	return b.String()
}

type Section struct {
	Name string

	tea.Model
}

func (s Section) Section() string { return s.Name }
