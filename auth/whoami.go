package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdAuthWhoami = cli.NewCmd[WhoAmI](CmdAuth, "whoami", func(cmd *cobra.Command) {})

type WhoAmI struct{}

func (c WhoAmI) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c *WhoAmI) runTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, models.WhoAmIHeader)
	drv.Activate(ctx, &models.WhoAmIChecker{})

	anc, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	res, err := anc.Get("")
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return errors.New("unexpected response")
	}

	var userInfo *api.Root
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return err
	}

	drv.Send(models.UserWhoAmIMsg(userInfo.Whoami))

	return nil
}
