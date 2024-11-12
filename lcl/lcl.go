package lcl

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/lcl/models"
	"github.com/anchordotdev/cli/trust"
	"github.com/anchordotdev/cli/truststore"
	"github.com/anchordotdev/cli/ui"
	"github.com/spf13/cobra"
)

var CmdLcl = cli.NewCmd[Command](cli.CmdRoot, "lcl", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization for lcl.host local development environment management.")
	cmd.Flags().StringVarP(&cfg.Lcl.RealmAPID, "realm", "r", cli.Defaults.Lcl.RealmAPID, "Realm for lcl.host local development environment management.")
	cmd.Flags().StringVarP(&cfg.Service.APID, "service", "s", cli.Defaults.Service.APID, "Service for lcl.host local development environment management.")

	// config
	cmd.Flags().StringVarP(&cfg.Lcl.Diagnostic.Addr, "addr", "a", cli.Defaults.Lcl.Diagnostic.Addr, "Address for local diagnostic web server.")

	// mkcert
	cmd.Flags().StringSliceVar(&cfg.Lcl.MkCert.Domains, "domains", cli.Defaults.Lcl.MkCert.Domains, "Domains to create certificate for.")
	cmd.Flags().StringVar(&cfg.Lcl.MkCert.SubCa, "subca", cli.Defaults.Lcl.MkCert.SubCa, "SubCA to create certificate for.")

	// setup
	cmd.Flags().StringVar(&cfg.Service.Category, "category", cli.Defaults.Service.Category, "Language or software type of the service.")
	cmd.Flags().StringVar(&cfg.Service.CertStyle, "cert-style", cli.Defaults.Service.CertStyle, "Provisioning method for lcl.host certificates.")

	// alias
	cmd.Flags().StringVar(&cfg.Service.Category, "language", cli.Defaults.Service.Category, "Language to integrate with Anchor.")
	_ = cmd.Flags().MarkDeprecated("language", "Please use `--category` instead.")
	cmd.Flags().StringVar(&cfg.Service.CertStyle, "method", cli.Defaults.Service.CertStyle, "Provisioning method for lcl.host certificates.")
	_ = cmd.Flags().MarkDeprecated("method", "Please use `--cert-style` instead.")
})

type Command struct {
	anc       *api.Session
	clipboard cli.Clipboard
}

func (c Command) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Command) run(ctx context.Context, drv *ui.Driver) error {
	if err := c.apiAuth(ctx, drv); err != nil {
		return err
	}

	drv.Activate(ctx, models.LclPreamble)

	drv.Activate(ctx, models.LclHeader)
	drv.Activate(ctx, models.LclHint)

	// run audit command
	drv.Activate(ctx, models.AuditHeader)
	drv.Activate(ctx, models.AuditHint)

	if err := c.systemConfig(ctx, drv); err != nil {
		return err
	}

	return c.appSetup(ctx, drv)
}

func (c *Command) apiAuth(ctx context.Context, drv *ui.Driver) error {
	cmd := &auth.Client{
		Anc:    c.anc,
		Hint:   models.LclSignInHint,
		Source: "lclhost",
	}

	var err error
	c.anc, err = cmd.Perform(ctx, drv)
	return err
}

func (c *Command) systemConfig(ctx context.Context, drv *ui.Driver) error {
	// audit diagnostic service

	userInfo, err := c.anc.UserInfo(ctx)
	if err != nil {
		return err
	}

	orgSlug := userInfo.PersonalOrg.Slug
	realmSlug := "localhost"

	localCAs, err := trust.FetchLocalDevCAs(ctx, c.anc)
	if err != nil {
		return err
	}

	personalCAs, err := trust.FetchExpectedCAs(ctx, c.anc, orgSlug, realmSlug)
	if err != nil {
		return err
	}

	// audit CA certs

	cmdAudit := &Audit{
		anc: c.anc,
		cas: localCAs,
	}

	auditInfo, err := cmdAudit.perform(ctx, drv)
	if err != nil {
		return err
	}

	isLocalhostCA := func(ca *truststore.CA) bool {
		for _, ca2 := range personalCAs {
			if ca.UniqueName == ca2.UniqueName {
				return true
			}
		}
		return false
	}

	switch {
	case len(auditInfo.Missing) == 0:
		drv.Activate(ctx, models.BootstrapSkip)

		return nil
	case slices.ContainsFunc(auditInfo.Missing, isLocalhostCA):
		drv.Activate(ctx, models.BootstrapHeader)
		drv.Activate(ctx, models.BootstrapHint)

		cmdBootstrap := &Bootstrap{
			anc:       c.anc,
			auditInfo: auditInfo,
		}

		return cmdBootstrap.perform(ctx, drv)
	default:
		drv.Activate(ctx, models.TrustHeader)
		drv.Activate(ctx, models.TrustHint)

		cmdTrust := &Trust{
			anc:       c.anc,
			auditInfo: auditInfo,
		}

		return cmdTrust.perform(ctx, drv)
	}
}

func (c *Command) appSetup(ctx context.Context, drv *ui.Driver) error {
	// run setup command
	drv.Activate(ctx, models.SetupHeader)
	drv.Activate(ctx, models.SetupHint)

	cmdSetup := &Setup{
		anc:       c.anc,
		clipboard: c.clipboard,
	}

	return cmdSetup.perform(ctx, drv)
}

func checkLoopbackDomain(ctx context.Context, drv *ui.Driver, domain string) error {
	drv.Activate(ctx, &models.DomainResolver{
		Domain: domain,
	})

	addrs, err := new(net.Resolver).LookupHost(ctx, domain)
	if err != nil {
		drv.Send(models.DomainStatusMsg(false))

		var dnserr *net.DNSError
		if errors.As(err, &dnserr) {
			return cli.UserError{Err: errors.New("no such host")}
		}

		return err
	}

	for _, addr := range addrs {
		if !slices.Contains(loopbackAddrs, addr) {
			drv.Send(models.DomainStatusMsg(false))

			return fmt.Errorf("%s domain resolved to non-loopback interface address: %s", domain, addr)
		}
	}
	drv.Send(models.DomainStatusMsg(true))

	return nil
}
