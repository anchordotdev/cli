package trust

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/component"
	componentmodels "github.com/anchordotdev/cli/component/models"
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
	Anc *api.Session

	OrgAPID, RealmAPID string
}

func (a Audit) UI() cli.UI {
	return cli.UI{
		RunTUI: a.RunTUI,
	}
}

func (c *Audit) RunTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	cmd := &auth.Client{
		Anc:    c.Anc,
		Source: "lclhost",
	}
	c.Anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.TrustAuditHeader)
	drv.Activate(ctx, models.TrustAuditHint)

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &truststoremodels.TrustStoreAudit{})

	expectedCAs, err := FetchExpectedCAs(ctx, c.Anc, orgAPID, realmAPID)
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

func (c *Audit) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgAPID != "" {
		return c.OrgAPID, nil
	}
	if cfg.Org.APID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Org.APID },
			Flag:          "--org",
			Singular:      "organization",
		})
		c.OrgAPID = cfg.Org.APID
		return c.OrgAPID, nil
	}

	selector := &component.Selector[api.Organization]{
		Prompt: "Which organization's env do you want to fetch?",
		Flag:   "--org",

		Fetcher: &component.Fetcher[api.Organization]{
			FetchFn: func() ([]api.Organization, error) { return c.Anc.GetOrgs(ctx) },
		},
	}

	org, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return org.Apid, nil
}

func (c *Audit) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}
	if cfg.Realm.APID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Realm.APID },
			Flag:          "--realm",
			Singular:      "realm",
		})
		c.RealmAPID = cfg.Realm.APID
		return c.RealmAPID, nil
	}

	selector := &component.Selector[api.Realm]{
		Prompt: fmt.Sprintf("Which %s realm's env do you want to fetch?", ui.Emphasize(orgAPID)),
		Flag:   "--realm",

		Fetcher: &component.Fetcher[api.Realm]{
			FetchFn: func() ([]api.Realm, error) { return c.Anc.GetOrgRealms(ctx, orgAPID) },
		},
	}

	realm, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return realm.Apid, nil
}
