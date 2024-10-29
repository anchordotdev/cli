package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	VersionHeader = ui.Section{
		Name: "VersionHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Show Version Info %s", ui.Whisper("`anchor version'"))),
		},
	}
)

type Version struct {
	Arch, Commit, Date, OS, Version string
}

func (m *Version) Init() tea.Cmd { return nil }

func (m *Version) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *Version) View() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s/%s) Commit: %s BuildDate: %s\n", m.Version, m.OS, m.Arch, m.Commit, m.Date)
	return b.String()
}
