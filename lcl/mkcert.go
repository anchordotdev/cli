package lcl

import (
	"context"
	"crypto/tls"
	"errors"
	"net/url"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/cert"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdLclMkCert = cli.NewCmd[MkCert](CmdLcl, "mkcert", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Args = cobra.NoArgs

	cmd.Flags().StringSliceVar(&cfg.Lcl.MkCert.Domains, "domains", []string{}, "Domains to create certificate for.")
	cmd.Flags().StringVar(&cfg.Lcl.MkCert.SubCa, "subca", "", "SubCA to create certificate for.")
})

type MkCert struct {
	anc *api.Session

	domains []string
	eab     *api.Eab

	chainSlug       string
	orgSlug         string
	realmSlug       string
	serviceSlug     string
	subCaSubjectUID string
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

	cmdCert := cert.Provision{
		Cert: tlsCert,
	}

	if err := cmdCert.RunTUI(ctx, drv, c.domains...); err != nil {
		return err
	}

	return nil
}

func (c *MkCert) perform(ctx context.Context, drv *ui.Driver) (*tls.Certificate, error) {
	cfg := cli.ConfigFromContext(ctx)

	var err error

	if c.chainSlug == "" {
		c.chainSlug = "ca"
	}

	if len(c.domains) == 0 {
		c.domains = cfg.Lcl.MkCert.Domains
		if len(c.domains) == 0 {
			return nil, errors.New("domains is required")
		}
	}

	if c.orgSlug == "" {
		userInfo, err := c.anc.UserInfo(ctx)
		if err != nil {
			return nil, err
		}
		c.orgSlug = userInfo.PersonalOrg.Slug
	}

	if c.realmSlug == "" {
		c.realmSlug = "localhost"
	}

	if c.serviceSlug == "" {
		c.serviceSlug = cfg.Lcl.Service
		if c.serviceSlug == "" {
			return nil, errors.New("service is required")
		}
	}

	if c.subCaSubjectUID == "" {
		c.subCaSubjectUID = cfg.Lcl.MkCert.SubCa
		if c.subCaSubjectUID == "" {
			return nil, errors.New("subca is required")
		}
	}

	c.eab, err = c.anc.CreateEAB(ctx, c.chainSlug, c.orgSlug, c.realmSlug, c.serviceSlug, c.subCaSubjectUID)
	if err != nil {
		return nil, err
	}

	acmeURL := cfg.AnchorURL + "/" + url.QueryEscape(c.orgSlug) + "/" + url.QueryEscape(c.realmSlug) + "/x509/" + c.chainSlug + "/acme"

	tlsCert, err := provisionCert(c.eab, c.domains, acmeURL)
	if err != nil {
		return nil, err
	}

	return tlsCert, nil
}
