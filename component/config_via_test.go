package component_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/MakeNowJust/heredoc"
	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/clitest"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestConfigVia(t *testing.T) {

	cmd := configViaCommand{}

	t.Run("env", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		t.Setenv("ORG", "env-org")
		cfg := cmdtest.Config(ctx)
		ctx = cli.ContextWithConfig(ctx, cfg)

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})

	t.Run("toml", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := new(cli.Config)
		cfg.Test.SystemFS = clitest.TestFS{
			"anchor.toml": &fstest.MapFile{
				Data: []byte(heredoc.Doc(`
        [org]
        apid = "toml-org"
        `)),
			},
		}
		if err := cfg.Load(ctx); err != nil {
			panic(err)
		}
		ctx = cli.ContextWithConfig(ctx, cfg)

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})

	t.Run("default", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := cmdtest.Config(ctx)
		ctx = cli.ContextWithConfig(ctx, cfg)

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})

	t.Run("flag", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := cmdtest.Config(ctx)
		cfg.Org.APID = "flag-org"
		ctx = cli.ContextWithConfig(ctx, cfg)

		uitest.TestTUIOutput(ctx, t, cmd.UI())
	})
}

type configViaCommand struct{}

func (c configViaCommand) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (*configViaCommand) run(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	if cfg.Org.APID != "" {
		drv.Activate(ctx, &models.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Org.APID },
			Flag:          "--org",
			Singular:      "organization",
		})
		return nil
	}
	if cfg.API.URL != "" {
		drv.Activate(ctx, &models.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.API.URL },
			Flag:          "API_URL=",
			Singular:      "api url",
		})
	}

	return nil
}
