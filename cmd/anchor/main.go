package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/google/go-github/v54/github"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/trust"
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
			&cli.Command{
				Name:  "auth",
				Use:   "auth <subcommand>",
				Short: "Authentication",
				Group: "user",

				SubCommands: []*cli.Command{
					&cli.Command{
						TUI: auth.SignIn{Config: cfg}.TUI(),

						Name:  "signin",
						Use:   "signin",
						Short: "Authenticate with your account",
						Long: heredoc.Doc(`
							Sign into your Anchor account for your local system user.

							Generate a new Personal Access Token (PAT) and store it in the system keychain
							for the local system user.
						`),
					},
					&cli.Command{
						TUI: auth.SignOut{Config: cfg}.TUI(),

						Name:  "signout",
						Use:   "signout",
						Short: "Invalidate your local Anchor account session",
						Long: heredoc.Doc(`
							Sign out of your Anchor account for your local system user.

							Remove your Personal Access Token (PAT) from the system keychain for your local
							system user.
						`),
					},
					&cli.Command{
						TUI: auth.WhoAmI{Config: cfg}.TUI(),

						Name:  "whoami",
						Use:   "whoami",
						Short: "Identify current account",
						Long: heredoc.Doc(`
							Print the details of the Anchor account for your local system user.
						`),
					},
				},
			},
			&cli.Command{
				TUI: trust.Command{Config: cfg}.TUI(),

				Name:  "trust",
				Use:   "trust [org[/realm[/ca]]]",
				Short: "Trust store",

				Long: heredoc.Doc(`
					Install the AnchorCA certificates of a target organization, realm, or CA into
					your local system's trust store. The default target is the localhost realm of
					your personal organization.

					After installation of the AnchorCA certificates, Leaf certificates under the
					AnchorCA certificates will be trusted by browsers and programs on your system.
				`),
			},
		},

		Preflight: versionCheck,
	}

	cfg = new(cli.Config)

	// Version info set by GoReleaser via ldflags

	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := cmd.Execute(ctx, cfg); err != nil {
		os.Exit(1)
	}
}

func versionCheck(ctx context.Context) error {
	if version == "dev" {
		return nil
	}

	release, _, err := github.NewClient(nil).Repositories.GetLatestRelease(ctx, "anchordotdev", "cli")
	if err != nil {
		return err
	}
	if release.TagName == nil || *release.TagName != "v"+version {
		return fmt.Errorf("Anchor CLI v%s is out of date, run `brew upgrade anchordotdev/tap/anchor` to install the latest", version)
	}
	return nil
}
