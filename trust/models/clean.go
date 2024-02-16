package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

type CleanPreflight struct {
	CertStates, TrustStores []string

	step preflightStep

	handle                 string
	expectedCAs, targetCAs []*truststore.CA

	spinner spinner.Model
}

func (c *CleanPreflight) Init() tea.Cmd {
	c.spinner = ui.Spinner()

	return c.spinner.Tick
}

func (c *CleanPreflight) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  var cmd tea.Cmd
  c.spinner, cmd = c.spinner.Update(msg)
  return c, cmd
}

func (c *CleanPreflight) View() string {
	states := strings.Join(c.CertStates, ", ")
	stores := strings.Join(c.TrustStores, ", ")

	var b strings.Builder
	fmt.Fprintln(&b, ui.Hint(fmt.Sprintf("Removing %s CA certificates from the %s store(s).", states, stores)))

	return b.String()
}

type cleanCAStep int

const (
	confirmingCA cleanCAStep = iota
	cleaningStores
	finishedCleanCA
)

type CleanCA struct {
	CA        *truststore.CA
	ConfirmCh chan<- struct{}

	step cleanCAStep

	stores  []truststore.Store
	cleaned map[truststore.Store]struct{}

	spinner spinner.Model
}

func (c *CleanCA) Init() tea.Cmd {
	c.spinner = ui.Spinner()

	c.cleaned = make(map[truststore.Store]struct{})

	return c.spinner.Tick
}

type (
	CleaningStoreMsg struct {
		truststore.Store
	}

	CleanedStoreMsg struct {
		truststore.Store
	}
)

func (c *CleanCA) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CleanedStoreMsg:
		c.cleaned[msg.Store] = struct{}{}
		return c, nil
	case CleaningStoreMsg:
		c.stores, c.step = append(c.stores, msg.Store), cleaningStores
		return c, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if c.ConfirmCh == nil {
				return c, nil
			}

			close(c.ConfirmCh)
			c.ConfirmCh = nil
		}
	}

	var cmd tea.Cmd
	c.spinner, cmd = c.spinner.Update(msg)
	return c, cmd
}

func (c *CleanCA) View() string {
	commonName := c.CA.Subject.CommonName
	serial := c.CA.SerialNumber.Text(16) // TODO: format serial as XXXX:XXXX:XXXX:XXXX
	algo := c.CA.PublicKeyAlgorithm

	var b strings.Builder
	fmt.Fprintln(&b, ui.Header(fmt.Sprintf("Remove \"%s\" (%s) %s Certificate", ui.Underline(commonName), ui.Whisper(serial), algo)))
	fmt.Fprintln(&b, ui.StepAlert(fmt.Sprintf("%s to remove the certificate (%s)", ui.Action("Press Enter"), ui.Accentuate("may require sudo"))))

	for _, store := range c.stores {
		if _, ok := c.cleaned[store]; ok {
			fmt.Fprintln(&b, ui.StepDone(fmt.Sprintf("removed certificate from the %s store.", ui.Emphasize(store.Description()))))
		} else {
			fmt.Fprintln(&b, ui.StepInProgress(fmt.Sprintf("removing certificate from the %s store…%s", ui.Emphasize(store.Description()), c.spinner.View())))
		}
	}

	return b.String()
}

type CleanEpilogue struct {
	Count int
}

func (CleanEpilogue) Init() tea.Cmd { return nil }

func (c CleanEpilogue) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return c, tea.Quit
}

func (c CleanEpilogue) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, ui.Header("Finished"))
	fmt.Fprintf(&b, ui.Hint("%d certificates were removed!\n"), c.Count)

	return b.String()
}
