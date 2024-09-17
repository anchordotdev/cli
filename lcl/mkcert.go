package lcl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/component"
	componentmodels "github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdLclMkCert = cli.NewCmd[MkCert](CmdLcl, "mkcert", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization to create certificate for.")
	cmd.Flags().StringVarP(&cfg.Lcl.RealmAPID, "realm", "r", cli.Defaults.Lcl.RealmAPID, "Realm to create certificate for.")
	cmd.Flags().StringVarP(&cfg.Service.APID, "service", "s", cli.Defaults.Service.APID, "Service to create certificate for.")

	cmd.Flags().StringSliceVar(&cfg.Lcl.MkCert.Domains, "domains", cli.Defaults.Lcl.MkCert.Domains, "Domains to create certificate for.")
})

type MkCert struct {
	anc *api.Session

	eab         *api.Eab
	Domains     []string
	OrgAPID     string
	RealmAPID   string
	ServiceAPID string

	// optional

	ChainAPID string
	SubCaAPID string
}

func (c MkCert) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *MkCert) run(ctx context.Context, drv *ui.Driver) error {
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

	tlsCert, err := c.perform(ctx, cfg, drv)
	if err != nil {
		return err
	}

	domains, err := c.domains(ctx, cfg)
	if err != nil {
		return err
	}

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return err
	}

	serviceAPID, err := c.serviceAPID(ctx, cfg, drv, orgAPID, realmAPID)
	if err != nil {
		return err
	}

	cmdCertProvision := cert.Provision{
		Cert:        tlsCert,
		Domains:     domains,
		OrgAPID:     orgAPID,
		RealmAPID:   realmAPID,
		ServiceAPID: serviceAPID,
	}

	return cmdCertProvision.Perform(ctx, drv)
}

func (c *MkCert) perform(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (*tls.Certificate, error) {
	chainAPID := c.ChainAPID
	if chainAPID == "" {
		chainAPID = "ca"
	}

	domains, err := c.domains(ctx, cfg)
	if err != nil {
		return nil, err
	}

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return nil, err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return nil, err
	}

	serviceAPID, err := c.serviceAPID(ctx, cfg, drv, orgAPID, realmAPID)
	if err != nil {
		return nil, err
	}

	subCaAPID, err := c.subcaAPID(ctx, cfg, orgAPID, realmAPID, chainAPID, serviceAPID)
	if err != nil {
		return nil, err
	}

	c.eab, err = c.anc.CreateEAB(ctx, chainAPID, orgAPID, realmAPID, serviceAPID, subCaAPID)
	if err != nil {
		return nil, err
	}

	acmeURL := cfg.AcmeURL(orgAPID, realmAPID, chainAPID)

	tlsCert, err := provisionCert(c.eab, domains, acmeURL)
	if err != nil {
		return nil, err
	}

	return tlsCert, nil
}

func (c *MkCert) domains(ctx context.Context, cfg *cli.Config) ([]string, error) {
	if len(c.Domains) != 0 {
		return c.Domains, nil
	}

	c.Domains = cfg.Lcl.MkCert.Domains
	if len(c.Domains) == 0 {
		return nil, cli.UserError{Err: errors.New("domains is required")}
	}
	return c.Domains, nil
}

func (c *MkCert) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
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
		Prompt: "Which organization's certificates do you want to create?",
		Flag:   "--org",

		Fetcher: &component.Fetcher[api.Organization]{
			FetchFn: func() ([]api.Organization, error) { return c.anc.GetOrgs(ctx) },
		},
	}

	org, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	c.OrgAPID = org.Apid
	return c.OrgAPID, nil
}

func (c *MkCert) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}

	if cfg.Lcl.RealmAPID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Lcl.RealmAPID },
			Flag:          "--realm",
			Singular:      "realm",
		})
		c.RealmAPID = cfg.Lcl.RealmAPID
		return c.RealmAPID, nil
	}

	selector := &component.Selector[api.Realm]{
		Prompt: fmt.Sprintf("Which %s realm's certificates do you want to create?", ui.Emphasize(orgAPID)),
		Flag:   "--realm",

		Fetcher: &component.Fetcher[api.Realm]{
			FetchFn: func() ([]api.Realm, error) { return c.anc.GetOrgRealms(ctx, orgAPID) },
		},
	}

	realm, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	c.RealmAPID = realm.Apid
	return c.RealmAPID, nil
}

func (c *MkCert) serviceAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID string) (string, error) {
	if c.ServiceAPID != "" {
		return c.ServiceAPID, nil
	}

	if cfg.Service.APID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Service.APID },
			Flag:          "--service",
			Singular:      "service",
		})
		c.ServiceAPID = cfg.Service.APID
		return c.ServiceAPID, nil
	}

	selector := &component.Selector[api.Service]{
		Prompt: fmt.Sprintf("Which %s/%s service's certificates do you want to create?", ui.Emphasize(orgAPID), ui.Emphasize(realmAPID)),
		Flag:   "--service",

		Fetcher: &component.Fetcher[api.Service]{
			FetchFn: func() ([]api.Service, error) { return c.anc.GetOrgServices(ctx, orgAPID, api.NonDiagnosticServices) },
		},
	}

	service, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", nil
	}
	c.ServiceAPID = service.Slug
	return c.ServiceAPID, nil
}

func (c *MkCert) subcaAPID(ctx context.Context, cfg *cli.Config, orgAPID, realmAPID, chainAPID, serviceAPID string) (string, error) {
	if c.SubCaAPID != "" {
		return c.SubCaAPID, nil
	}

	if cfg.Lcl.MkCert.SubCa != "" {
		c.SubCaAPID = cfg.Lcl.MkCert.SubCa
		return cfg.Lcl.MkCert.SubCa, nil
	}

	attachments, err := c.anc.GetServiceAttachments(ctx, orgAPID, serviceAPID)
	if err != nil {
		return "", err
	}

	for _, a := range attachments {
		if a.Relationships.Realm.Apid == realmAPID && a.Relationships.Chain.Apid == chainAPID {
			c.SubCaAPID = cfg.Lcl.MkCert.SubCa
			return *a.Relationships.SubCa.Apid, nil
		}
	}

	return "", cli.UserError{
		Err: fmt.Errorf("invalid org, realm, and service combination"),
	}
}
