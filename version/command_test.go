package version

import (
	"context"
	"flag"
	"runtime"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
)

var (
	_ = flag.Bool("update", false, "ignored")
)

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
