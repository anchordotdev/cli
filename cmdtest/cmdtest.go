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

func TestCfg(t *testing.T, cmd *cobra.Command, args ...string) *cli.Config {
	cmd = cli.NewTestCmd(cmd)
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true
	if err := envdecode.Decode(cfg); err != nil && err != envdecode.ErrNoTargetFieldsAreSet {
		t.Fatal(err)
	}

	_, err := execute(cmd, args...)
	require.NoError(t, err)

	return cfg
}

func TestCmd(t *testing.T, cmd *cobra.Command, args ...string) *cobra.Command {
	_, err := executeSkip(cmd, args...)
	require.NoError(t, err)

	return cmd
}

func TestError(t *testing.T, cmd *cobra.Command, args ...string) error {
	_, err := executeSkip(cmd, args...)
	require.Error(t, err)

	return err
}

func TestHelp(t *testing.T, cmd *cobra.Command, args ...string) {
	root := cmd.Root()

	b, err := execute(root, args...)
	require.NoError(t, err)

	out, err := io.ReadAll(b)
	require.NoError(t, err)

	teatest.RequireEqualOutput(t, out)
}

func execute(cmd *cobra.Command, args ...string) (*bytes.Buffer, error) {
	b := new(bytes.Buffer)
	cmd.SetErr(b)
	cmd.SetOut(b)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return b, err
}

func executeSkip(cmd *cobra.Command, args ...string) (*bytes.Buffer, error) {
	cmd = cli.NewTestCmd(cmd)
	cfg := cli.ConfigFromCmd(cmd)
	cfg.Test.SkipRunE = true

	return execute(cmd, args...)
}
