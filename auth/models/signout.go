package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type SignOutHeader struct{}

func (SignOutHeader) Init() tea.Cmd { return nil }

func (m *SignOutHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SignOutHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Signout from Anchor.dev %s", ui.Whisper("`anchor auth signout`"))))
	return b.String()
}

type SignOutSuccess struct{}

func (SignOutSuccess) Init() tea.Cmd { return nil }

func (m *SignOutSuccess) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SignOutSuccess) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.StepDone("Signed out."))
	return b.String()
}
