package trust

import (
	"context"
	"os"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/trust/models"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdTrustClean = cli.NewCmd[Clean](CmdTrust, "clean", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Args = cobra.NoArgs

	cmd.Flags().StringSliceVar(&cfg.Trust.Clean.States, "cert-states", []string{"expired"}, "Cert states to clean.")
	cmd.Flags().StringVarP(&cfg.Trust.Org, "organization", "o", "", "Organization to trust.")
	cmd.Flags().BoolVar(&cfg.Trust.NoSudo, "no-sudo", false, "Disable sudo prompts.")
	cmd.Flags().StringVarP(&cfg.Trust.Realm, "realm", "r", "", "Realm to trust.")
	cmd.Flags().StringSliceVar(&cfg.Trust.Stores, "trust-stores", []string{"homebrew", "nss", "system"}, "Trust stores to update.")

	cmd.Hidden = true
})

type Clean struct {
	Anc                *api.Session
	OrgSlug, RealmSlug string
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

	drv.Activate(ctx, &models.TrustCleanHeader{})
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

func (c Clean) Perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	if c.OrgSlug == "" && c.RealmSlug == "" {
		c.OrgSlug, c.RealmSlug, err = fetchOrgAndRealm(ctx, c.Anc)
		if err != nil {
			return err
		}
	}

	drv.Activate(ctx, &models.TrustCleanAudit{})

	expectedCAs, err := fetchExpectedCAs(ctx, c.Anc, c.OrgSlug, c.RealmSlug)
	if err != nil {
		return err
	}

	stores, sudoMgr, err := loadStores(cfg)
	if err != nil {
		return err
	}

	sudoMgr.AroundSudo = func(sudo func()) {
		unpausec := drv.Pause()
		defer close(unpausec)

		sudo()
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
				return err
			}

			drv.Send(models.CACleanedMsg{Store: store})
		}
	}

	return nil
}
