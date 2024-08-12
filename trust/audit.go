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

	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization to trust.")
	cmd.Flags().StringVarP(&cfg.Realm.APID, "realm", "r", cli.Defaults.Realm.APID, "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", cli.Defaults.Trust.Stores, "Trust stores to update.")

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

	drv.Activate(ctx, models.TrustAuditHeader)
	drv.Activate(ctx, models.TrustAuditHint)

	drv.Activate(ctx, &truststoremodels.TrustStoreAudit{})

	org, realm, err := fetchOrgAndRealm(ctx, c.anc)
	if err != nil {
		return err
	}

	expectedCAs, err := FetchExpectedCAs(ctx, c.anc, org, realm)
	if err != nil {
		return err
	}

	stores, err := LoadStores(ctx, nil)
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
