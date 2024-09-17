package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfigVia struct {
	Config        *cli.Config
	ConfigFetchFn cli.ConfigFetchFunc

	Flag, Singular string
}

func (m *ConfigVia) Init() tea.Cmd { return nil }

func (m *ConfigVia) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *ConfigVia) View() string {
	var b strings.Builder

	source := m.Config.ViaSource(m.ConfigFetchFn)
	value := fmt.Sprintf("%+v", m.ConfigFetchFn(m.Config))

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Using %s %s from %s. %s",
		ui.Emphasize(value),
		m.Singular,
		source,
		ui.Whisper(fmt.Sprintf("You can also use `%s %s`.", m.Flag, value)),
	)))

	return b.String()
}
