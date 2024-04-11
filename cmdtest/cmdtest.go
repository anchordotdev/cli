package cmdtest

import (
	"bytes"
	"io"
	"testing"

	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T, cmd *cobra.Command, args ...string) {
	b := new(bytes.Buffer)
	cmd.SetErr(b)
	cmd.SetOut(b)
	cmd.SetArgs(args)

	err := cmd.Execute()
	require.NoError(t, err)

	out, err := io.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}

	teatest.RequireEqualOutput(t, out)
}
