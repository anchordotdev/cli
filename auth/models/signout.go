package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type SignOutPreamble struct{}

func (SignOutPreamble) Init() tea.Cmd { return nil }

func (m *SignOutPreamble) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *SignOutPreamble) View() string {
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
