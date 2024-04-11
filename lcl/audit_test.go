package lcl

import (
	"context"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/cmdtest"
	"github.com/anchordotdev/cli/ui/uitest"
)

func TestCmdDefLclAudit(t *testing.T) {
	cmd := CmdAuthLclAudit
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true
	root := cmd.Root()

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, root, "lcl", "audit", "--help")
	})
}

func TestAudit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty support on windows")
	}

	if srv.IsProxy() {
		t.Skip("lcl audit unsupported in proxy mode")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := new(cli.Config)
	cfg.API.URL = srv.URL
	cfg.Trust.Stores = []string{"mock"}
	var err error
	if cfg.API.Token, err = srv.GeneratePAT("anky@anchor.dev"); err != nil {
		t.Fatal(err)
	}
	ctx = cli.ContextWithConfig(ctx, cfg)

	t.Run("basics", func(t *testing.T) {
		uitest.TestTUIOutput(ctx, t, new(Audit).UI())
	})
}
