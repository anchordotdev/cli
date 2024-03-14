package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type LclCleanHeader struct{}

func (LclCleanHeader) Init() tea.Cmd { return nil }

func (m *LclCleanHeader) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *LclCleanHeader) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Clean lcl.host CA Certificates from Local Trust Store(s) %s", ui.Whisper("`anchor trust clean`"))))
	return b.String()
}

type LclCleanHint struct {
	TrustStores []string

	spinner spinner.Model
}

func (c *LclCleanHint) Init() tea.Cmd {
	c.spinner = ui.WaitingSpinner()

	return c.spinner.Tick
}

func (c *LclCleanHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	c.spinner, cmd = c.spinner.Update(msg)
	return c, cmd
}

func (c *LclCleanHint) View() string {
	stores := strings.Join(c.TrustStores, ", ")

	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint(fmt.Sprintf("Removing lcl.host CA certificates from the %s store(s).", stores)))

	return b.String()
}
