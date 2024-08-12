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

var CmdLclTrust = cli.NewCmd[Trust](CmdLcl, "trust", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().BoolVar(&cfg.Trust.NoSudo, "no-sudo", false, "Disable sudo prompts.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", []string{"homebrew", "nss", "system"}, "Trust stores to update.")
})

type Trust struct {
	anc *api.Session

	auditInfo *truststore.AuditInfo
}

func (c Trust) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Trust) run(ctx context.Context, drv *ui.Driver) error {
	// TODO: convert flags to fields on c

	var err error
	if c.anc, err = new(auth.Client).Perform(ctx, drv); err != nil {
		return err
	}

	drv.Activate(ctx, models.TrustHeader)
	drv.Activate(ctx, models.TrustHint)

	return c.perform(ctx, drv)
}

func (c *Trust) perform(ctx context.Context, drv *ui.Driver) error {
	if c.auditInfo == nil {
		var err error
		if c.auditInfo, err = c.performAudit(ctx, drv); err != nil {
			return err
		}
	}

	cmdTrust := &trust.Command{
		Anc:       c.anc,
		AuditInfo: c.auditInfo,
	}

	return cmdTrust.Perform(ctx, drv)
}

func (c *Trust) performAudit(ctx context.Context, drv *ui.Driver) (*truststore.AuditInfo, error) {
	drv.Activate(ctx, &truststoreModels.TrustStoreAudit{})

	stores, err := trust.LoadStores(ctx, drv)
	if err != nil {
		return nil, err
	}

	cas, err := trust.FetchLocalDevCAs(ctx, c.anc)
	if err != nil {
		return nil, err
	}

	auditInfo, err := trust.PerformAudit(ctx, stores, cas)
	if err != nil {
		return nil, err
	}

	drv.Send(truststoreModels.AuditInfoMsg(auditInfo))

	return auditInfo, nil
}
