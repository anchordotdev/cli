package org

import (
	"context"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/org/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdOrgCreate = cli.NewCmd[Create](CmdOrg, "create", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVar(&cfg.Org.Name, "org-name", "", "Name for created org.")
})

type Create struct {
	Anc *api.Session

	OrgName string
}

func (c Create) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Create) run(ctx context.Context, drv *ui.Driver) error {
	var err error
	clientCmd := &auth.Client{
		Anc: c.Anc,
	}
	c.Anc, err = clientCmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.OrgCreateHeader{})
	drv.Activate(ctx, &models.OrgCreateHint{})

	_, err = c.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Create) Perform(ctx context.Context, drv *ui.Driver) (*api.Organization, error) {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	c.OrgName, err = c.orgName(ctx, cfg, drv)
	if err != nil {
		return nil, err
	}
	drv.Activate(ctx, &models.CreateOrgSpinner{})

	org, err := c.Anc.CreateOrg(ctx, c.OrgName)
	if err != nil {
		return nil, err
	}

	drv.Send(ui.HideModelsMsg{
		Models: []string{"CreateOrgNameInput", "CreateOrgSpinner"},
	})

	drv.Activate(ctx, &models.CreateOrgResult{
		Org: *org,
	})

	return org, nil
}

func (c *Create) orgName(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgName != "" {
		return c.OrgName, nil
	}
	if cfg.Org.Name != "" {
		c.OrgName = cfg.Org.Name
		return c.OrgName, nil
	}

	if c.OrgName == "" {
		inputc := make(chan string)
		drv.Activate(ctx, &models.CreateOrgNameInput{
			InputCh: inputc,
		})

		select {
		case orgName := <-inputc:
			c.OrgName = orgName
			return c.OrgName, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return "", nil
}
