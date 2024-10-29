package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	VersionUpgradeHeader = ui.Section{
		Name: "VersionUpgradeHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Check for Upgrade %s", ui.Whisper("`anchor version upgrade`"))),
		},
	}

	VersionUpgradeUnavailable = ui.Section{
		Name: "VersionUpgradeUnavailable",
		Model: ui.MessageLines{
			ui.StepAlert("Already up to date!"),
			ui.StepHint("Your anchor CLI is already up to date, check back soon for updates."),
		},
	}
)

type VersionUpgrade struct {
	Command     string
	InClipboard bool
}

func (m *VersionUpgrade) Init() tea.Cmd { return nil }

func (m *VersionUpgrade) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m *VersionUpgrade) View() string {
	var b strings.Builder

	if m.InClipboard {
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Copied %s to your clipboard.", ui.Announce(m.Command))))
	}

	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s `%s` to update to the latest version.", ui.Action("Run"), ui.Emphasize(m.Command))))
	fmt.Fprintln(&b, ui.StepHint(fmt.Sprintf("Not using homebrew? Explore other options here: %s", ui.URL("https://github.com/anchordotdev/cli"))))

	return b.String()
}
