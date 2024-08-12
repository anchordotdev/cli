package lcl

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/truststore"
	truststoreModels "github.com/anchordotdev/cli/truststore/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdLclAudit = cli.NewCmd[Audit](CmdLcl, "audit", func(cmd *cobra.Command) {})

type Audit struct {
	anc *api.Session

	cas []*truststore.CA
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

	drv.Activate(ctx, models.AuditHeader)
	drv.Activate(ctx, models.AuditHint)

	_, err = c.perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c Audit) perform(ctx context.Context, drv *ui.Driver) (*truststore.AuditInfo, error) {
	drv.Activate(ctx, &truststoreModels.TrustStoreAudit{})

	stores, err := trust.LoadStores(ctx, drv)
	if err != nil {
		return nil, err
	}

	cas := c.cas
	if len(cas) == 0 {
		var err error
		if cas, err = trust.FetchLocalDevCAs(ctx, c.anc); err != nil {
			return nil, err
		}
	}

	auditInfo, err := trust.PerformAudit(ctx, stores, cas)
	if err != nil {
		return nil, err
	}

	drv.Send(truststoreModels.AuditInfoMsg(auditInfo))

	return auditInfo, nil
}
