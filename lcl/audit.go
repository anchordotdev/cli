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

var CmdLclAudit = cli.NewCmd[Audit](CmdLcl, "audit", func(cmd *cobra.Command) {
	cmd.Args = cobra.NoArgs
})

type Audit struct {
	anc                *api.Session
	orgSlug, realmSlug string
}

func (c Audit) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c Audit) run(ctx context.Context, drv *ui.Driver) error {
	var err error
	cmd := &auth.Client{
		Anc:    c.anc,
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.AuditHeader{})
	drv.Activate(ctx, &models.AuditHint{})

	_, err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

type LclAuditResult struct {
	diagnosticServiceExists, trusted bool
}

func (c Audit) perform(ctx context.Context, drv *ui.Driver) (*LclAuditResult, error) {
	var result = LclAuditResult{}

	drv.Activate(ctx, &models.AuditResources{})

	if c.orgSlug == "" {
		userInfo, err := c.anc.UserInfo(ctx)
		if err != nil {
			return nil, err
		}
		c.orgSlug = userInfo.PersonalOrg.Slug
	}

	if c.realmSlug == "" {
		c.realmSlug = "localhost"
	}

	var diagnosticService *api.Service
	services, err := c.anc.GetOrgServices(ctx, c.orgSlug)
	if err != nil {
		return nil, err
	}
	for _, service := range services {
		if service.ServerType == "diagnostic" {
			diagnosticService = &service
		}
	}

	if diagnosticService == nil {
		drv.Send(models.AuditResourcesNotFoundMsg{})
	} else {
		result.diagnosticServiceExists = true
		drv.Send(models.AuditResourcesFoundMsg{})
	}

	drv.Activate(ctx, &models.AuditTrust{})

	// FIXME: use config anc?
	trustStoreAuditResult, err := trust.PerformAudit(ctx, c.anc, c.orgSlug, c.realmSlug)
	if err != nil {
		return nil, err
	}

	missingCount := len(trustStoreAuditResult.Missing)
	result.trusted = missingCount == 0
	drv.Send(models.AuditTrustMissingMsg(missingCount))

	// TODO: audit local app status

	return &result, nil
}
