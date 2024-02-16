package models

import (
	"fmt"
	"strings"

	"github.com/anchordotdev/cli/ui"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type SignInPreamble struct {
	Message string
}

func (SignInPreamble) Init() tea.Cmd { return nil }

func (m SignInPreamble) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m SignInPreamble) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Signin to Anchor.dev %s", ui.Whisper("`anchor auth signin`"))))
	if m.Message != "" {
		fmt.Fprintln(&b, m.Message)
	}

	return b.String()
}

type SignInPrompt struct {
	ConfirmCh       chan<- struct{}
	InClipboard     bool
	UserCode        string
	VerificationURL string
}

func (SignInPrompt) Init() tea.Cmd { return nil }

func (m *SignInPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.ConfirmCh != nil {
				close(m.ConfirmCh)
				m.ConfirmCh = nil
			}
		}
	}

	return m, nil
}

func (m *SignInPrompt) View() string {
	var b strings.Builder

	if m.ConfirmCh != nil {
		if m.InClipboard {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Copied your user code %s to your clipboard.", ui.Emphasize(m.UserCode))))
		} else {
			fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("Copy your user code: %s", ui.Announce(m.UserCode))))
		}
		fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to open %s in your browser", ui.Action("Press Enter"), ui.URL(m.VerificationURL))))
		return b.String()
	}

	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Copied your user code to your clipboard.")))
	fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Opened %s in your browser", ui.URL(m.VerificationURL))))

	return b.String()
}

type SignInChecker struct {
	whoami string

	spinner spinner.Model
}

func (m *SignInChecker) Init() tea.Cmd {
	m.spinner = ui.Spinner()

	return m.spinner.Tick
}

type UserSignInMsg string

func (m *SignInChecker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case UserSignInMsg:
		m.whoami = string(msg)
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SignInChecker) View() string {
	var b strings.Builder
	if m.whoami == "" {
		fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("Signing in… %s", m.spinner.View())))
	} else {
		fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("Signed in as %s.", ui.Emphasize(m.whoami))))
	}
	return b.String()
}
