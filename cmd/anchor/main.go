package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/google/go-github/v54/github"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl"
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
			{
				Name:  "auth",
				Use:   "auth <subcommand>",
				Short: "Authentication",
				Group: "user",

				SubCommands: []*cli.Command{
					{
						UI: auth.SignIn{Config: cfg}.UI(),

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
						UI: auth.SignOut{Config: cfg}.UI(),

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
						UI: auth.WhoAmI{Config: cfg}.UI(),

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
				UI: lcl.Command{Config: cfg}.UI(),

				Name:   "lcl",
				Use:    "lcl <subcommand>",
				Short:  "lcl.host",
				Hidden: true,

				SubCommands: []*cli.Command{
					{
						UI: lcl.Audit{Config: cfg}.UI(),

						Name:  "audit",
						Use:   "audit",
						Short: "Audit lcl.host HTTPS Local Development Environment",
					},
					{
						UI: lcl.LclConfig{Config: cfg}.UI(),

						Name:  "config",
						Use:   "config",
						Short: "Configure System for lcl.host Local Development",
					},
					{
						UI: lcl.MkCert{Config: cfg}.UI(),

						Name:  "mkcert",
						Use:   "mkcert",
						Short: "Provision Certificate for lcl.host Local Development",
					},
					{
						UI: lcl.Setup{Config: cfg}.UI(),

						Name:  "setup",
						Use:   "setup",
						Short: "Setup lcl.host Application",
					},
				},
			},
			{
				UI: trust.Command{Config: cfg}.UI(),

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

				SubCommands: []*cli.Command{
					{
						UI: trust.Audit{Config: cfg}.UI(),

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
						UI: trust.Clean{Config: cfg}.UI(),

						Name:   "clean",
						Use:    "clean TODO",
						Short:  "clean the Local trust store(s)",
						Hidden: true,

						Long: heredoc.Doc(`
						TODO
						`),
					},
				},
			},
		},

		Preflight: versionCheck,
	}

	cfg = new(cli.Config)

	// Version info set by GoReleaser via ldflags

	version = "dev"
	commit  = "none"    //lint:ignore U1000 set by GoReleaser
	date    = "unknown" //lint:ignore U1000 set by GoReleaser
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
		return fmt.Errorf("anchor CLI v%s is out of date, run `brew upgrade anchordotdev/tap/anchor` to install the latest", version)
	}
	return nil
}
