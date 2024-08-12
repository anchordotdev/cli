package lcl

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdLclAudit(t *testing.T) {
	t.Run("--help", func(t *testing.T) {
		cmdtest.TestHelp(t, CmdLclAudit, "lcl", "audit", "--help")
	})
}

func TestAudit(t *testing.T) {
	if srv.IsProxy() {
		t.Skip("lcl audit unsupported in proxy mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
		"/v0/orgs/org-slug/realms": {
			Example: "development",
		},
	}
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		uitest.TestTUIOutput(ctx, t, new(Audit).UI())
	})

	t.Run("missing-localhost-ca", func(t *testing.T) {
		cfg.Test.Prefer = map[string]cli.ConfigTestPrefer{
			"/v0/orgs": {
				Example: "anky_personal",
			},
			"/v0/orgs/ankydotdev/realms": {
				Example: "anky_personal",
			},
			"/v0/orgs/ankydotdev/realms/localhost/x509/credentials": {
				Example: "anky_personal",
			},
		}
		ctx = cli.ContextWithConfig(ctx, cfg)

		uitest.TestTUIOutput(ctx, t, new(Audit).UI())
	})
}
