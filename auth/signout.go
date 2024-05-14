package auth

import (
	"context"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/keyring"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdAuthSignout = cli.NewCmd[SignOut](CmdAuth, "signout", func(cmd *cobra.Command) {})

type SignOut struct{}

func (s SignOut) UI() cli.UI {
	return cli.UI{
		RunTUI: s.runTUI,
	}
}

func (s *SignOut) runTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, models.SignOutHeader)

	kr := keyring.Keyring{Config: cfg}
	err := kr.Delete(keyring.APIToken)

	if err == nil {
		drv.Activate(ctx, models.SignOutSuccess)
	}

	return err
}
