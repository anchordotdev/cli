package cmdtest

import (
	"bytes"
	"io"
	"testing"

	"github.com/anchordotdev/cli"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/joeshaw/envdecode"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T, cmd *cobra.Command, args ...string) *bytes.Buffer {
	// pull from env again, since setting in tests is too late for normal env handling
	cfg := cli.ConfigFromCmd(cmd)
	if err := envdecode.Decode(cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		panic(err)
	}

	root := cmd.Root()

	b := new(bytes.Buffer)
	root.SetErr(b)
	root.SetOut(b)
	root.SetArgs(args)

	err := root.Execute()
	require.NoError(t, err)

	return b
}

func TestOutput(t *testing.T, cmd *cobra.Command, args ...string) {
	b := TestExecute(t, cmd, args...)

	out, err := io.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}
