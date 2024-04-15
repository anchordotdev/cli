package version

import (
	"context"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api/apitest"
	"github.com/anchordotdev/cli/cmdtest"
)

func TestCmdVersion(t *testing.T) {
	cmd := CmdVersion
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	t.Run("--help", func(t *testing.T) {
		cmdtest.TestOutput(t, cmd, "version", "--help")
	})
}

func TestCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no pty support on windows")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := new(Command)
	buf, err := apitest.RunTTY(ctx, cmd.UI())
	if err != nil {
		t.Fatal(err)
	}

	if want, got := String()+"\n", buf.String(); want != got {
		t.Errorf("want output %q, got %q", want, got)
	}
}
