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
	header = lipgloss.NewStyle().Bold(true)
	hint   = lipgloss.NewStyle().Faint(true).SetString("|")

	Header    = header.SetString("\n#").Render
	Skip      = header.Faint(true).SetString("\n# Skipped:").Render
	Hint      = hint.Render
	Underline = lipgloss.NewStyle().Underline(true).Render
	Warning   = header.SetString(bgBanana(fgMidnight("!")) + fgBanana(" Warning:")).Render

	// https://github.com/charmbracelet/lipgloss/blob/v0.9.1/style.go#L149

	StepAlert      = lipgloss.NewStyle().SetString("    " + Announce("!")).Render
	StepDone       = lipgloss.NewStyle().SetString("    -").Render
	StepHint       = hint.SetString("    |").Render
	StepNext       = hint.SetString("    -").Render
	StepInProgress = lipgloss.NewStyle().SetString("    *").Render
	StepPrompt     = lipgloss.NewStyle().SetString("    " + Prompt.Render("?")).Render
	StepWarning    = header.SetString("    " + bgBanana(fgMidnight("!")) + fgBanana(" Warning:")).Render

	Accentuate         = lipgloss.NewStyle().Italic(true).Render
	Action             = lipgloss.NewStyle().Bold(true).Foreground(colorBrandPrimary).Render
	Announce           = lipgloss.NewStyle().Background(colorBrandSecondary).Render
	Danger             = lipgloss.NewStyle().Bold(true).Foreground(colorDanger).Render
	Emphasize          = lipgloss.NewStyle().Bold(true).Render
	EmphasizeUnderline = lipgloss.NewStyle().Bold(true).Underline(true).Render
	Titlize            = lipgloss.NewStyle().Bold(true).Render
	URL                = lipgloss.NewStyle().Faint(true).Underline(true).Render
	Whisper            = lipgloss.NewStyle().Faint(true).Render

	bgBanana = lipgloss.NewStyle().Background(colorBanana).Render

	fgBanana   = lipgloss.NewStyle().Foreground(colorBanana).Render
	fgMidnight = lipgloss.NewStyle().Foreground(colorMidnight).Render

	colorBrandPrimary   = colorMandarin
	colorBrandSecondary = colorGrape
	colorDanger         = colorApple

	// Brand Palette
	colorMidnight = lipgloss.Color("#110C18")
	colorGrape    = lipgloss.Color("#60539E")
	colorMandarin = lipgloss.Color("#FF6000")
	colorApple    = lipgloss.Color("#CE4433")
	colorBanana   = lipgloss.Color("#FBF5AC")
	// colorLime = lipgloss.Color("#9EC756")

	Prompt = lipgloss.NewStyle().Foreground(colorBrandPrimary)

	Waiting = spinner.MiniDot
)

func WaitingSpinner() spinner.Model {
	return spinner.New(
		spinner.WithSpinner(Waiting),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colorBrandSecondary)),
	)
}

type ListItem[T any] struct {
	Key    string
	String string
	Value  T
}

func (li ListItem[T]) FilterValue() string { return li.Key }

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
		fmt.Fprint(w, Action(fmt.Sprintf("    > %s", i.String)))
	} else {
		fmt.Fprintf(w, "      %s", i.String)
	}
}

func List[T any](items []ListItem[T]) list.Model {
	var lis []list.Item
	for _, item := range items {
		lis = append(lis, item)
	}

	l := list.New(lis, itemDelegate[T]{}, 80, len(items))
	l.InfiniteScrolling = true
	l.SetShowFilter(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)

	return l
}

func Domains(domains []string) string {
	var styledDomains []string

	for _, domain := range domains {
		styledDomains = append(styledDomains, URL(domain))
	}

	return strings.Join(styledDomains, ", ")
}
