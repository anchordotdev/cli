package trust

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	truststoremodels "github.com/anchordotdev/cli/truststore/models"
	"github.com/anchordotdev/cli/ui"
)

var CmdTrustAudit = cli.NewCmd[Audit](CmdTrust, "audit", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Trust.Org, "org", "o", "", "Organization to trust.")
	cmd.Flags().StringVarP(&cfg.Trust.Realm, "realm", "r", "", "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", []string{"homebrew", "nss", "system"}, "Trust stores to update.")

	cmd.MarkFlagsRequiredTogether("org", "realm")
})

type Audit struct {
	anc *api.Session
}

func (a Audit) UI() cli.UI {
	return cli.UI{
		RunTUI: a.RunTUI,
	}
}

func (c *Audit) RunTUI(ctx context.Context, drv *ui.Driver) error {
	var err error
	cmd := &auth.Client{
		Anc:    c.anc,
		Source: "lclhost",
	}
	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	cfg := cli.ConfigFromContext(ctx)

	drv.Activate(ctx, &models.TrustAuditHeader{})
	drv.Activate(ctx, &models.TrustAuditHint{})

	drv.Activate(ctx, &truststoremodels.TrustStoreAudit{})

	org, realm, err := fetchOrgAndRealm(ctx, c.anc)
	if err != nil {
		return err
	}

	expectedCAs, err := fetchExpectedCAs(ctx, c.anc, org, realm)
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

	auditInfo, err := audit.Perform()
	if err != nil {
		return err
	}

	drv.Send(truststoremodels.AuditInfoMsg(auditInfo))

	drv.Activate(ctx, &models.TrustAuditInfo{
		AuditInfo: auditInfo,
		Stores:    stores,
	})

	return nil
}
