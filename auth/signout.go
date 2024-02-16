package auth

import (
	"context"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/keyring"
	"github.com/anchordotdev/cli/ui"
)

type SignOut struct {
	Config *cli.Config
}

func (s SignOut) UI() cli.UI {
	return cli.UI{
		RunTUI: s.runTUI,
	}
}

func (s *SignOut) runTUI(ctx context.Context, drv *ui.Driver) error {
	drv.Activate(ctx, &models.SignOutPreamble{})

	kr := keyring.Keyring{Config: s.Config}
	err := kr.Delete(keyring.APIToken)

	if err == nil {
		drv.Activate(ctx, &models.SignOutSuccess{})
	}

	return err
}
