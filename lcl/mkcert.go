package lcl

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/component"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdLclMkCert = cli.NewCmd[MkCert](CmdLcl, "mkcert", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Lcl.Org, "org", "o", "", "Organization to create certificate for.")
	cmd.Flags().StringVarP(&cfg.Lcl.Realm, "realm", "r", "", "Realm to create certificate for.")
	cmd.Flags().StringVarP(&cfg.Lcl.Service, "service", "s", "", "Service to create certificate for.")

	cmd.Flags().StringSliceVar(&cfg.Lcl.MkCert.Domains, "domains", []string{}, "Domains to create certificate for.")
})

type MkCert struct {
	anc *api.Session

	domains     []string
	eab         *api.Eab
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

	tlsCert, err := c.perform(ctx, drv)
	if err != nil {
		return err
	}

	cmdCertProvision := cert.Provision{
		Cert:        tlsCert,
		Domains:     c.domains,
		OrgAPID:     c.OrgAPID,
		RealmAPID:   c.RealmAPID,
		ServiceAPID: c.ServiceAPID,
	}

	return cmdCertProvision.Perform(ctx, drv)
}

func (c *MkCert) perform(ctx context.Context, drv *ui.Driver) (*tls.Certificate, error) {
	cfg := cli.ConfigFromContext(ctx)

	chainAPID := c.ChainAPID
	if chainAPID == "" {
		chainAPID = "ca"
	}

	if len(c.domains) == 0 {
		c.domains = cfg.Lcl.MkCert.Domains
		if len(c.domains) == 0 {
			return nil, cli.UserError{Err: errors.New("domains is required")}
		}
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

	acmeURL := cfg.AnchorURL + "/" + url.QueryEscape(orgAPID) + "/" + url.QueryEscape(realmAPID) + "/x509/" + chainAPID + "/acme"

	tlsCert, err := provisionCert(c.eab, c.domains, acmeURL)
	if err != nil {
		return nil, err
	}

	return tlsCert, nil
}

func (c *MkCert) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgAPID != "" {
		return c.OrgAPID, nil
	}
	if cfg.Lcl.Org != "" {
		return cfg.Lcl.Org, nil
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
	return org.Apid, nil
}

func (c *MkCert) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}
	if cfg.Lcl.Realm != "" {
		return cfg.Lcl.Realm, nil
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
	return realm.Apid, nil
}

func (c *MkCert) serviceAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID string) (string, error) {
	if c.ServiceAPID != "" {
		return c.ServiceAPID, nil
	}
	if cfg.Lcl.Service != "" {
		return cfg.Lcl.Service, nil
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
	return service.Slug, nil
}

func (c *MkCert) subcaAPID(ctx context.Context, cfg *cli.Config, orgAPID, realmAPID, chainAPID, serviceAPID string) (string, error) {
	if c.SubCaAPID != "" {
		return c.SubCaAPID, nil
	}
	if cfg.Lcl.MkCert.SubCa != "" {
		return cfg.Lcl.MkCert.SubCa, nil
	}

	attachments, err := c.anc.GetServiceAttachments(ctx, orgAPID, serviceAPID)
	if err != nil {
		return "", err
	}

	for _, a := range attachments {
		if a.Relationships.Realm.Apid == realmAPID && a.Relationships.Chain.Apid == chainAPID {
			return *a.Relationships.SubCa.Apid, nil
		}
	}

	return "", cli.UserError{
		Err: fmt.Errorf("invalid org, realm, and service combination"),
	}
}
