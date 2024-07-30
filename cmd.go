package cli

import (
	"context"
	"errors"

	"github.com/MakeNowJust/heredoc"
	"github.com/anchordotdev/cli/stacktrace"
	"github.com/anchordotdev/cli/ui"
	"github.com/joeshaw/envdecode"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/cobra"
)

type CmdDef struct {
	Name string

	Use   string
	Args  cobra.PositionalArgs
	Short string
	Long  string

	SubDefs []CmdDef
}

var rootDef = CmdDef{
	Name: "anchor",

	Use:   "anchor <command> <subcommand> [flags]",
	Args:  cobra.NoArgs,
	Short: "anchor - a CLI interface for anchor.dev",
	Long: heredoc.Doc(`
		anchor is a command line interface for the Anchor certificate management platform.

		It provides a developer friendly interface for certificate management.
	`),

	SubDefs: []CmdDef{
		{
			Name: "auth",

			Use:   "auth [flags]",
			Args:  cobra.NoArgs,
			Short: "Manage Anchor.dev Authentication",

			SubDefs: []CmdDef{
				{
					Name: "signin",

					Use:   "signin [flags]",
					Args:  cobra.NoArgs,
					Short: "Authenticate With Your Account",
					Long: heredoc.Doc(`
						Sign into your Anchor account for your local system user.

						Generate a new Personal Access Token (PAT) and store it in the system keychain
						for the local system user.
					`),
				},
				{
					Name: "signout",

					Use:   "signout [flags]",
					Args:  cobra.NoArgs,
					Short: "Invalidate Local Anchor Session",
					Long: heredoc.Doc(`
						Sign out of your Anchor account for your local system user.

						Remove your Personal Access Token (PAT) from the system keychain for your local
						system user.
					`),
				},
				{
					Name: "whoami",

					Use:   "whoami [flags]",
					Args:  cobra.NoArgs,
					Short: "Identify Current Anchor.dev Account",
					Long: heredoc.Doc(`
						Print the details of the Anchor account for your local system user.
					`),
				},
			},
		},
		{
			Name: "lcl",

			Use:   "lcl [flags]",
			Args:  cobra.NoArgs,
			Short: "Manage lcl.host Local Development Environment",

			SubDefs: []CmdDef{
				{
					Name: "audit",

					Use:   "audit [flags]",
					Args:  cobra.NoArgs,
					Short: "Audit lcl.host HTTPS Local Development Environment",
				},
				{
					Name: "clean",

					Use:   "clean [flags]",
					Args:  cobra.NoArgs,
					Short: "Clean lcl.host CA Certificates from the Local Trust Store(s)",
				},
				{
					Name: "config",

					Use:   "config [flags]",
					Args:  cobra.NoArgs,
					Short: "Configure System for lcl.host Local Development",
				},
				{
					Name: "env",

					Use:   "env [flags]",
					Args:  cobra.NoArgs,
					Short: "Generate lcl.host ENV",
				},
				{
					Name: "mkcert",

					Use:   "mkcert [flags]",
					Args:  cobra.NoArgs,
					Short: "Provision Certificate for lcl.host Local Development",
				},
				{
					Name: "setup",

					Use:   "setup [flags]",
					Args:  cobra.NoArgs,
					Short: "Setup lcl.host Application",
				},
			},
		},
		{
			Name: "service",

			Use:   "service [flags]",
			Args:  cobra.NoArgs,
			Short: "Manage services",
			SubDefs: []CmdDef{
				{
					Name: "env",

					Use:   "env [flags]",
					Args:  cobra.NoArgs,
					Short: "Fetch Environment Variables for Service",
				},
			},
		},
		{
			Name: "trust",

			Use:   "trust [flags]",
			Args:  cobra.NoArgs,
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
					Args:  cobra.NoArgs,
					Short: "Audit CA Certificates in Your Local Trust Store(s)",
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
					Args:  cobra.NoArgs,
					Short: "Clean CA Certificates from your Local Trust Store(s)",
				},
			},
		},
		{
			Name: "version",

			Use:   "version",
			Args:  cobra.NoArgs,
			Short: "Show version info",
		},
	},
}

var cmdDefByCommands = map[*cobra.Command]*CmdDef{}

var constructorByCommands = map[*cobra.Command]func() *cobra.Command{}

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

	constructor := func() *cobra.Command {
		cfg := new(Config)
		defaults.SetDefaults(cfg)
		if err := envdecode.Decode(cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
			panic(err)
		}

		cmd := &cobra.Command{
			Use:          def.Use,
			Args:         def.Args,
			Short:        def.Short,
			Long:         def.Long,
			SilenceUsage: true,
		}

		ctx := ContextWithConfig(context.Background(), cfg)
		cmd.SetContext(ctx)

		cmd.SetErrPrefix(ui.Danger("Error!"))

		fn(cmd)

		cmd.RunE = func(cmd *cobra.Command, args []string) (returnedError error) {
			cfg := ConfigFromCmd(cmd)
			if cfg.Test.SkipRunE {
				return nil
			}

			var t T

			switch any(t).(type) {
			case ShowHelp:
				cmd.HelpFunc()(cmd, args)
				return nil
			}

			ctx, cancel := context.WithCancelCause(cmd.Context())
			defer cancel(nil)

			drv, prg := ui.NewDriverTUI(ctx)
			defer func() {
				// release/restore
				drv.Program.Quit()
				drv.Program.Wait()
			}()

			errc := make(chan error)

			go func() {
				defer close(errc)

				_, err := prg.Run()
				cancel(err)

				errc <- err
			}()

			if err := stacktrace.CapturePanic(func() error { return t.UI().RunTUI(ctx, drv) }); err != nil {
				var uierr ui.Error
				if errors.As(err, &uierr) {
					drv.Activate(context.Background(), uierr.Model)
					err = uierr.Err
				}

				if err != nil && isReportable(err) {
					ReportError(ctx, err, drv, cmd, args)
				}
				return err
			}

			drv.Program.Quit()
			if err := <-errc; err != nil {
				if isReportable(err) {
					ReportError(ctx, err, drv, cmd, args)
				}
				return err
			}

			return nil
		}

		new(cmdWrapper).wrap(cmd)

		return cmd
	}

	cmd := constructor()
	if parent != nil {
		parent.AddCommand(cmd)
	}
	cmdDefByCommands[cmd] = def
	constructorByCommands[cmd] = constructor
	return cmd
}

func NewTestCmd(cmd *cobra.Command) *cobra.Command {
	return constructorByCommands[cmd]()
}

type cmdWrapper struct {
	err error
}

func (w *cmdWrapper) wrap(cmd *cobra.Command) {
	if fn := cmd.PersistentPreRunE; fn != nil {
		cmd.PersistentPreRunE = w.runFunc(fn)
	}
	if fn := cmd.PreRunE; fn != nil {
		cmd.PreRunE = w.runFunc(fn)
	}
	if fn := cmd.RunE; fn != nil {
		cmd.RunE = w.runFunc(fn)
	}
	if fn := cmd.PostRunE; fn != nil {
		cmd.PostRunE = w.runFunc(fn)
	}
	if fn := cmd.PersistentPostRunE; fn != nil {
		cmd.PersistentPostRunE = w.runFunc(fn)
	}
}

func (w *cmdWrapper) runFunc(fn func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if w.err != nil {
			return w.err
		}

		w.err = fn(cmd, args)
		return w.err
	}
}
