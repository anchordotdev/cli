package trust

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/component"
	componentmodels "github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdTrustClean = cli.NewCmd[Clean](CmdTrust, "clean", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringSliceVar(&cfg.Trust.Clean.States, "cert-states", cli.Defaults.Trust.Clean.States, "Cert states to clean.")
	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization to trust.")
	cmd.Flags().BoolVar(&cfg.Trust.NoSudo, "no-sudo", cli.Defaults.Trust.NoSudo, "Disable sudo prompts.")
	cmd.Flags().StringVarP(&cfg.Realm.APID, "realm", "r", cli.Defaults.Realm.APID, "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", cli.Defaults.Trust.Stores, "Trust stores to update.")

	cmd.MarkFlagsRequiredTogether("org", "realm")

	cmd.Hidden = true
})

type Clean struct {
	Anc                *api.Session
	OrgAPID, RealmAPID string
}

func (c Clean) UI() cli.UI {
	return cli.UI{
		RunTUI: c.runTUI,
	}
}

func (c *Clean) runTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	cmd := &auth.Client{
		Anc: c.Anc,
	}
	c.Anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.TrustCleanHeader)
	drv.Activate(ctx, &models.TrustCleanHint{
		CertStates:  cfg.Trust.Clean.States,
		TrustStores: cfg.Trust.Stores,
	})

	err = c.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Clean) Perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.TrustCleanAudit{})

	expectedCAs, err := FetchExpectedCAs(ctx, c.Anc, orgAPID, realmAPID)
	if err != nil {
		return err
	}

	stores, err := LoadStores(ctx, drv)
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

	targetCAs := info.AllCAs(cfg.Trust.Clean.States...)
	drv.Send(targetCAs)

	tmpDir, err := os.MkdirTemp("", "anchor-trust-clean")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, ca := range targetCAs {
		confirmc := make(chan struct{})
		drv.Activate(ctx, &models.TrustCleanCA{
			CA:        ca,
			Config:    cfg,
			ConfirmCh: confirmc,
		})

		if !cfg.NonInteractive {
			select {
			case <-confirmc:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		if ca.FilePath == "" {
			if err := writeCAFile(ca, tmpDir); err != nil {
				return err
			}
		}

		for _, store := range stores {
			if !info.IsPresent(ca, store) {
				continue
			}

			drv.Send(models.CACleaningMsg{Store: store})

			if _, err := store.UninstallCA(ca); err != nil {
				return classifyError(err)
			}

			drv.Send(models.CACleanedMsg{Store: store})
		}
	}

	return nil
}

func classifyError(err error) error {
	// TODO: should these be a ui.Error's?
	switch {
	case strings.HasSuffix(err.Error(), "sudo: 3 incorrect password attempts"):
		return cli.UserError{
			Err: fmt.Errorf("sudo failed: invalid password, please try again with the correct password."),
		}
	case strings.HasSuffix(strings.TrimSpace(err.Error()), "SecTrustSettingsRemoveTrustSettings: The authorization was canceled by the user."):
		return cli.UserError{
			Err: fmt.Errorf("remove cert failed: action canceled."),
		}
	default:
		return err
	}
}

func (c *Clean) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
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

func (c *Clean) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
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
