package lcl

import (
	"context"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdLclClean = cli.NewCmd[LclClean](CmdLcl, "clean", func(cmd *cobra.Command) {})

type LclClean struct {
	anc *api.Session
}

func (c LclClean) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c LclClean) run(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	clientCmd := &auth.Client{
		Anc: c.anc,
	}
	c.anc, err = clientCmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	cfg.Trust.Clean.States = []string{"all"}

	orgAPID, err := c.personalOrgAPID(ctx)
	if err != nil {
		return err
	}

	realmAPID, err := c.localhostRealmAPID()
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.LclCleanHeader)
	drv.Activate(ctx, &models.LclCleanHint{
		TrustStores: cfg.Trust.Stores,
	})

	cmd := &trust.Clean{
		Anc:       c.anc,
		OrgAPID:   orgAPID,
		RealmAPID: realmAPID,
	}

	err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c LclClean) personalOrgAPID(ctx context.Context) (string, error) {
	userInfo, err := c.anc.UserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.PersonalOrg.Slug, nil
}

func (c LclClean) localhostRealmAPID() (string, error) {
	return "localhost", nil
}
