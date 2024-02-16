package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	hint = lipgloss.NewStyle().Faint(true).SetString("|")

	Header    = lipgloss.NewStyle().Bold(true).SetString("#").Render
	Hint      = hint.Copy().Render
	Underline = lipgloss.NewStyle().Underline(true).Render

	// https://github.com/charmbracelet/lipgloss/blob/v0.9.1/style.go#L149

	StepAlert      = lipgloss.NewStyle().SetString("    " + Announce("!")).Render
	StepDone       = lipgloss.NewStyle().SetString("    -").Render
	StepHint       = hint.Copy().SetString("    |").Render
	StepInProgress = lipgloss.NewStyle().SetString("    *").Render
	StepPrompt     = lipgloss.NewStyle().SetString("    " + Prompt.Render("?")).Render

	Accentuate = lipgloss.NewStyle().Italic(true).Render
	Action     = lipgloss.NewStyle().Bold(true).Foreground(colorBrandPrimary).Render
	Announce   = lipgloss.NewStyle().Background(colorBrandSecondary).Render
	Emphasize  = lipgloss.NewStyle().Bold(true).Render
	Titlize    = lipgloss.NewStyle().Bold(true).Render
	URL        = lipgloss.NewStyle().Faint(true).Underline(true).Render
	Whisper    = lipgloss.NewStyle().Faint(true).Render

	colorBrandPrimary   = lipgloss.Color("#ff6000")
	colorBrandSecondary = lipgloss.Color("#7000ff")

	Prompt = lipgloss.NewStyle().Foreground(colorBrandPrimary)
)

func Spinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorBrandSecondary)),
	)
}

type ListItem[T any] struct {
	Key   T
	Value string
}

func (li ListItem[T]) FilterValue() string { return li.Value }

type itemDelegate[T any] struct{}

func (d itemDelegate[T]) Height() int                             { return 1 }
func (d itemDelegate[T]) Spacing() int                            { return 0 }
func (d itemDelegate[T]) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate[T]) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(ListItem[T])
	if !ok {
		return
	}

	if index == m.Index() {
		fmt.Fprintf(w, Action(fmt.Sprintf("    > %s", i.Value)))
	} else {
		fmt.Fprintf(w, fmt.Sprintf("      %s", i.Value))
	}
}

func List[T any](items []ListItem[T]) list.Model {
	var lis []list.Item
	for _, item := range items {
		lis = append(lis, item)
	}

	l := list.New(lis, itemDelegate[T]{}, 80, len(items))
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	return l
}

func Domains(domains []string) string {
	var styled_domains []string

	for _, domain := range domains {
		styled_domains = append(styled_domains, URL(domain))
	}

	return strings.Join(styled_domains, ", ")
}
