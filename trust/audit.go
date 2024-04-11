package trust

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/truststore"
)

type Audit struct{}

func (a Audit) UI() cli.UI {
	return cli.UI{
		RunTTY: a.run,
	}
}

func (a *Audit) run(ctx context.Context, tty termenv.File) error {
	cfg := cli.ConfigFromContext(ctx)

	anc, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	org, realm, err := fetchOrgAndRealm(ctx, anc)
	if err != nil {
		return err
	}

	expectedCAs, err := fetchExpectedCAs(ctx, anc, org, realm)
	if err != nil {
		return err
	}

	stores, _, err := loadStores(cfg)
	if err != nil {
		return err
	}

	audit := &truststore.Audit{
		Expected: expectedCAs,
		Stores:   stores,
		SelectFn: checkAnchorCert,
	}

	info, err := audit.Perform()
	if err != nil {
		return err
	}

	for _, ca := range info.Valid {
		fmt.Fprintf(tty, "%s (%s) %-7s \"%s\"\n",
			boldGreen.Render(fmt.Sprintf("%-8s", "VALID")),
			period(ca),
			ca.PublicKeyAlgorithm,
			commonName(ca),
		)

		fmt.Fprintln(tty)

		for _, store := range stores {
			fmt.Fprintf(tty, "%7s %35s    %s\n", "", store.Description(), boldGreen.Render("TRUSTED"))
		}

		fmt.Fprintln(tty)
	}

	for _, ca := range info.Missing {
		fmt.Fprintf(tty, "%s (%s) %-7s \"%s\"\n",
			boldRed.Render(fmt.Sprintf("%-8s", "MISSING")),
			period(ca),
			ca.PublicKeyAlgorithm,
			commonName(ca),
		)

		fmt.Fprintln(tty)

		for _, store := range stores {
			if info.IsPresent(ca, store) {
				fmt.Fprintf(tty, "%7s %35s    %s\n", "", store.Description(), boldGreen.Render("TRUSTED"))
			} else {
				fmt.Fprintf(tty, "%7s %35s    %s\n", "", store.Description(), boldRed.Render("NOT PRESENT"))
			}
		}

		fmt.Fprintln(tty)
	}

	for _, ca := range info.Extra {
		fmt.Fprintf(tty, "%s (%s) %-7s \"%s\"\n",
			faint.Render(fmt.Sprintf("%-8s", "EXTRA")),
			period(ca),
			ca.PublicKeyAlgorithm,
			commonName(ca),
		)

		fmt.Fprintln(tty)

		for _, store := range stores {
			if info.IsPresent(ca, store) {
				fmt.Fprintf(tty, "%7s %35s    %s\n", "", store.Description(), faintGreen.Render("TRUSTED"))
			} else {
				fmt.Fprintf(tty, "%7s %35s    %s\n", "", store.Description(), faint.Render("NOT PRESENT"))
			}
		}

		fmt.Fprintln(tty)
	}

	return nil
}

var (
	darkGreen  = lipgloss.Color("#008000")
	darkRed    = lipgloss.Color("#800000")
	lightGreen = lipgloss.Color("#00ff00")
	lightRed   = lipgloss.Color("#ff0000")

	boldGreen  = lipgloss.NewStyle().Bold(true).Foreground(lightGreen)
	boldRed    = lipgloss.NewStyle().Bold(true).Foreground(lightRed)
	faintGreen = lipgloss.NewStyle().Faint(true).Foreground(darkGreen)
	faintRed   = lipgloss.NewStyle().Faint(true).Foreground(darkRed)

	faint = lipgloss.NewStyle().Faint(true)

	italic    = lipgloss.NewStyle().Italic(true)
	underline = lipgloss.NewStyle().Underline(true)
)

func commonName(ca *truststore.CA) string {
	return underline.Render(fmt.Sprintf("%s", ca.Subject.CommonName))
}

func period(ca *truststore.CA) string {
	startAt := ca.NotBefore.Format("2006-01-02")
	expireAt := ca.NotAfter.Add(1 * time.Second).Format("2006-01-02")

	return italic.Render(fmt.Sprintf("%s - %s", startAt, expireAt))
}
