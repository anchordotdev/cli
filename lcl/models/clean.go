package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

var LclCleanHeader = ui.Section{
	Name: "LclCleanHeader",
	Model: ui.MessageLines{
		ui.Header(fmt.Sprintf("Clean lcl.host CA Certificates from Local Trust Store(s) %s", ui.Whisper("`anchor trust clean`"))),
	},
}

type LclCleanHint struct {
	TrustStores []string
}

func (c *LclCleanHint) Init() tea.Cmd { return nil }

func (c *LclCleanHint) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return c, nil }

func (c *LclCleanHint) View() string {
	stores := strings.Join(c.TrustStores, ", ")

	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint(fmt.Sprintf("We'll remove lcl.host CA certificates from the %s store(s).", stores)))

	return b.String()
}
