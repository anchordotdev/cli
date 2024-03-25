package version

import (
	"context"
	"testing"

	"github.com/anchordotdev/cli/api/apitest"
)

func TestCommand(t *testing.T) {
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
