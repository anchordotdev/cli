package cli

import (
	"context"

	"github.com/MakeNowJust/heredoc"
	"github.com/anchordotdev/cli/ui"
	"github.com/joeshaw/envdecode"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/cobra"
)

type CmdDef struct {
	Name string

	Use   string
	Short string
	Long  string

	SubDefs []CmdDef
}

var rootDef = CmdDef{
	Name: "anchor",

	Use:   "anchor <command> <subcommand> [flags]",
	Short: "anchor - a CLI interface for anchor.dev",
	Long: heredoc.Doc(`
		anchor is a command line interface for the Anchor certificate management platform.

		It provides a developer friendly interface for certificate management.
	`),

	SubDefs: []CmdDef{
		{
			Name: "auth",

			Use:   "auth <subcommand> [flags]",
			Short: "Manage Anchor.dev Authentication",

			SubDefs: []CmdDef{
				{
					Name: "signin",

					Use:   "signin [flags]",
					Short: "Authenticate with your account",
					Long: heredoc.Doc(`
						Sign into your Anchor account for your local system user.

						Generate a new Personal Access Token (PAT) and store it in the system keychain
						for the local system user.
					`),
				},
				{
					Name: "signout",

					Use:   "signout [flags]",
					Short: "Invalidate your local Anchor account session",
					Long: heredoc.Doc(`
						Sign out of your Anchor account for your local system user.

						Remove your Personal Access Token (PAT) from the system keychain for your local
						system user.
					`),
				},
				{
					Name: "whoami",

					Use:   "whoami [flags]",
					Short: "Identify current account",
					Long: heredoc.Doc(`
						Print the details of the Anchor account for your local system user.
					`),
				},
			},
		},
		{
			Name: "lcl",

			Use:   "lcl [flags]",
			Short: "Manage lcl.host Local Development Environment",

			SubDefs: []CmdDef{
				{
					Name: "audit",

					Use:   "audit [flags]",
					Short: "Audit lcl.host HTTPS Local Development Environment",
				},
				{
					Name: "clean",

					Use:   "clean [flags]",
					Short: "Clean lcl.host CA Certificates from the Local Trust Store(s)",
				},
				{
					Name: "config",

					Use:   "config [flags]",
					Short: "Configure System for lcl.host Local Development",
				},
				{
					Name: "mkcert",

					Use:   "mkcert [flags]",
					Short: "Provision Certificate for lcl.host Local Development",
				},
				{
					Name: "setup",

					Use:   "setup [flags]",
					Short: "Setup lcl.host Application",
				},
			},
		},
		{
			Name: "trust",

			Use:   "trust [flags]",
			Short: "Manage CA Certificates in your Local Trust Store(s)",
			Long: heredoc.Doc(`
				Install the AnchorCA certificates of a target organization, realm, or CA into
				your local system's trust store. The default target is the localhost realm of
				your personal organization.

				After installation of the AnchorCA certificates, Leaf certificates under the
				AnchorCA certificates will be trusted by browsers and programs on your system.
			`),
			SubDefs: []CmdDef{
				{
					Name: "audit",

					Use:   "audit [flags]",
					Short: "Audit CA Certificates in your Local Trust Store(s)",
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
					Name: "clean",

					Use:   "clean [flags]",
					Short: "Clean CA Certificates from your Local Trust Store(s)",
				},
			},
		},
		{
			Name: "version",

			Use:   "version",
			Short: "Show version info",
		},
	},
}

var cmdDefByCommands = map[*cobra.Command]*CmdDef{}

type UIer interface {
	UI() UI
}

func NewCmd[T UIer](parent *cobra.Command, name string, fn func(*cobra.Command)) *cobra.Command {
	var def, parentDef *CmdDef
	if parent != nil {
		var ok bool
		if parentDef, ok = cmdDefByCommands[parent]; !ok {
			panic("unregistered parent command")
		}
		for _, sub := range parentDef.SubDefs {
			if sub.Name == name {
				def = &sub
				break
			}
		}
		if def == nil {
			panic("missing subcommand definition")
		}
	} else {
		def = &rootDef
	}

	cfg := new(Config)
	defaults.SetDefaults(cfg)
	if err := envdecode.Decode(cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		panic(err)
	}

	cmd := &cobra.Command{
		Use:   def.Use,
		Short: def.Short,
		Long:  def.Long,
		// FIMXE: add PersistentPreRunE version check
	}

	if parent != nil {
		parent.AddCommand(cmd)
	}

	ctx := ContextWithConfig(context.Background(), cfg)
	cmd.SetContext(ctx)

	fn(cmd)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cfg := ConfigFromCmd(cmd)
		if cfg.Test.SkipRunE {
			return nil
		}

		ctx, cancel := context.WithCancelCause(cmd.Context())
		defer cancel(nil)

		drv, prg := ui.NewDriverTUI(ctx)
		errc := make(chan error)

		go func() {
			defer close(errc)

			_, err := prg.Run()
			cancel(err)

			errc <- err
		}()

		var t T
		if err := t.UI().RunTUI(ctx, drv); err != nil && err != context.Canceled {
			prg.Quit()

			<-errc // TODO: report this somewhere?
			return err
		}

		prg.Quit()

		return <-errc // TODO: special handling for a UI error
	}

	cmdDefByCommands[cmd] = def
	return cmd
}
