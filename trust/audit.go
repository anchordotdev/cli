package trust

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
)

var CmdTrustAudit = cli.NewCmd[Audit](CmdTrust, "audit", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Trust.Org, "organization", "o", "", "Organization to trust.")
	cmd.Flags().StringVarP(&cfg.Trust.Realm, "realm", "r", "", "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", []string{"homebrew", "nss", "system"}, "Trust stores to update.")

	cmd.MarkFlagsRequiredTogether("organization", "realm")
})

type Audit struct{}

func (a Audit) UI() cli.UI {
	return cli.UI{
		RunTUI: a.RunTUI,
	}
}

func (c *Audit) RunTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, &models.TrustAuditHeader{})
	drv.Activate(ctx, &models.TrustAuditScan{})

	anc, err := api.NewClient(cfg)
	if err != nil {
		return err
	}

	org, realm, err := fetchOrgAndRealm(ctx, anc)
	if err != nil {
		return err
	}

	expectedCAs, err := fetchExpectedCAs(ctx, anc, org, realm)
	if err != nil {
		return err
	}

	stores, _, err := loadStores(cfg)
	if err != nil {
		return err
	}

	audit := &truststore.Audit{
		Expected: expectedCAs,
		Stores:   stores,
		SelectFn: checkAnchorCert,
	}

	info, err := audit.Perform()
	if err != nil {
		return err
	}

	drv.Send(models.TrustAuditScanFinishedMsg(true))

	drv.Activate(ctx, &models.TrustAuditInfo{
		AuditInfo: info,
		Stores:    stores,
	})

	return nil
}
