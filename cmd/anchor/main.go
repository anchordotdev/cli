package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/atotto/clipboard"
	"github.com/google/go-github/v54/github"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/ui"
	versionpkg "github.com/anchordotdev/cli/version"
)

var (
	cmd = &cli.Command{
		Name:  "anchor",
		Use:   "anchor <command> <subcommand> [flags]",
		Short: "anchor - a CLI interface for anchor.dev",
		Long: heredoc.Doc(`
			anchor is a command line interface for the Anchor certificate management platform.

			It provides a developer friendly interface for certificate management.
		`),

		SubCommands: []*cli.Command{
			{
				Name:  "auth",
				Use:   "auth <subcommand>",
				Short: "Manage Anchor.dev Authentication",
				Group: "user",

				SubCommands: []*cli.Command{
					{
						UI: auth.SignIn{}.UI(),

						Name:  "signin",
						Use:   "signin",
						Short: "Authenticate with your account",
						Long: heredoc.Doc(`
							Sign into your Anchor account for your local system user.

							Generate a new Personal Access Token (PAT) and store it in the system keychain
							for the local system user.
						`),
					},
					{
						UI: auth.SignOut{}.UI(),

						Name:  "signout",
						Use:   "signout",
						Short: "Invalidate your local Anchor account session",
						Long: heredoc.Doc(`
							Sign out of your Anchor account for your local system user.

							Remove your Personal Access Token (PAT) from the system keychain for your local
							system user.
						`),
					},
					{
						UI: auth.WhoAmI{}.UI(),

						Name:  "whoami",
						Use:   "whoami",
						Short: "Identify current account",
						Long: heredoc.Doc(`
							Print the details of the Anchor account for your local system user.
						`),
					},
				},
			},
			{
				UI: lcl.Command{}.UI(),

				Name:  "lcl",
				Use:   "lcl <subcommand>",
				Short: "Manage lcl.host Local Development Environment",

				SubCommands: []*cli.Command{
					{
						UI: lcl.Audit{}.UI(),

						Name:  "audit",
						Use:   "audit",
						Short: "Audit lcl.host HTTPS Local Development Environment",
					},
					{
						UI: lcl.LclClean{}.UI(),

						Name:  "clean",
						Use:   "clean",
						Short: "Clean lcl.host CA Certificates from the Local Trust Store(s)",
					},
					{
						UI: lcl.LclConfig{}.UI(),

						Name:  "config",
						Use:   "config",
						Short: "Configure System for lcl.host Local Development",
					},
					{
						UI: lcl.MkCert{}.UI(),

						Name:  "mkcert",
						Use:   "mkcert",
						Short: "Provision Certificate for lcl.host Local Development",
					},
					{
						UI: lcl.Setup{}.UI(),

						Name:  "setup",
						Use:   "setup",
						Short: "Setup lcl.host Application",
					},
				},
			},
			{
				UI: trust.Command{}.UI(),

				Name:  "trust",
				Use:   "trust [org[/realm[/ca]]]",
				Short: "Manage Local Trust Stores",

				Long: heredoc.Doc(`
					Install the AnchorCA certificates of a target organization, realm, or CA into
					your local system's trust store. The default target is the localhost realm of
					your personal organization.

					After installation of the AnchorCA certificates, Leaf certificates under the
					AnchorCA certificates will be trusted by browsers and programs on your system.
				`),

				SubCommands: []*cli.Command{
					{
						UI: trust.Audit{}.UI(),

						Name:  "audit",
						Use:   "audit [org[/realm[/ca]]]",
						Short: "Local trust store audit",

						Long: heredoc.Doc(`
							Perform an audit of the local trust store(s) and report any expected, missing,
							or extra CA certificates per store. A set of expected CAs is fetched for the
							target org and (optional) realm. The default stores to audit are system, nss,
							and homebrew.

							CA certificate states:

								* VALID:   an expected CA certificate is present in every trust store.
								* MISSING: an expected CA certificate is missing in one or more stores.
								* EXTRA:   an unexpected CA certificate is present in one or more stores.
						`),
					},
					{
						UI: trust.Clean{}.UI(),

						Name:   "clean",
						Use:    "clean TODO",
						Short:  "Clean CA Certificates from the Local Trust Store(s)",
						Hidden: true,
					},
				},
			},
			{
				UI: versionpkg.Command{}.UI(),

				Name:  "version",
				Use:   "version",
				Short: "Show version info",
			},
		},

		Preflight: versionCheck,
	}

	// Version info set by GoReleaser via ldflags

	version, commit, date string
)

func init() {
	if version != "" {
		versionpkg.Set(version, commit, date)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}

func versionCheck(ctx context.Context) error {
	if version == "" {
		return nil
	}

	release, _, err := github.NewClient(nil).Repositories.GetLatestRelease(ctx, "anchordotdev", "cli")
	if err != nil {
		return nil
	}
	if publishedAt := release.PublishedAt.GetTime(); publishedAt != nil && time.Since(*publishedAt).Hours() < 24 {
		return nil
	}

	if release.TagName == nil || *release.TagName != "v"+version {
		fmt.Println(ui.StepHint("A new release of the anchor CLI is available."))
		command := "brew update && brew upgrade anchor"
		if err := clipboard.WriteAll(command); err == nil {
			fmt.Println(ui.StepAlert(fmt.Sprintf("Copied %s to your clipboard.", ui.Announce(command))))
		}
		fmt.Println(ui.StepAlert(fmt.Sprintf("%s `%s` to update to the latest version.", ui.Action("Run"), ui.Emphasize(command))))
		fmt.Println(ui.StepHint(fmt.Sprintf("Not using homebrew? Explore other options here: %s", ui.URL("https://github.com/anchordotdev/cli"))))
		fmt.Println()
	}
	return nil
}
