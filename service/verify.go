package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/component"
	componentmodels "github.com/anchordotdev/cli/component/models"
	"github.com/anchordotdev/cli/service/models"
	"github.com/anchordotdev/cli/ui"
)

var cmdServiceVerify = cli.NewCmd[Verify](CmdService, "verify", func(cmd *cobra.Command) {
	cfg := cli.ConfigFromCmd(cmd)

	cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization of the service to verify.")
	cmd.Flags().StringVarP(&cfg.Realm.APID, "realm", "r", cli.Defaults.Realm.APID, "Realm instance of the service to verify.")
	cmd.Flags().StringVarP(&cfg.Service.APID, "service", "s", cli.Defaults.Service.APID, "Service to verify.")

	cmd.Flags().DurationVar(&cfg.Service.Verify.Timeout, "timeout", cli.Defaults.Service.Verify.Timeout, "Time to wait for a successful verification of the service.")
})

type Verify struct {
	anc *api.Session

	OrgAPID, RealmAPID, ServiceAPID string
}

func (c Verify) UI() cli.UI {
	return cli.UI{
		RunTUI: c.RunTUI,
	}
}

func (c *Verify) RunTUI(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	var err error
	cmd := &auth.Client{
		Anc: c.anc,
	}

	drv.Activate(ctx, models.VerifyHeader)
	drv.Activate(ctx, models.VerifyHint)

	c.anc, err = cmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	orgAPID, err := c.orgAPID(ctx, cfg, drv)
	if err != nil {
		return err
	}

	serviceAPID, err := c.serviceAPID(ctx, cfg, drv, orgAPID)
	if err != nil {
		return err
	}

	// TODO: show "fetching" model
	attachments, err := c.anc.GetServiceAttachments(ctx, orgAPID, serviceAPID)
	if err != nil {
		return err
	}

	realmAPID, err := c.realmAPID(ctx, cfg, drv, orgAPID, serviceAPID, attachments)
	if err != nil {
		return err
	}

	idx := slices.IndexFunc(attachments, func(attachment api.Attachment) bool {
		return attachment.Relationships.Realm.Apid == realmAPID
	})
	if idx == -1 {
		panic("no attachment for --realm")
	}
	attachment := attachments[idx]

	// TODO: show "fetching" model

	credentials, err := c.anc.GetCredentials(ctx, orgAPID, realmAPID, api.SubCA(*attachment.Relationships.SubCa.Apid))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Service.Verify.Timeout)
	defer cancel()

	addrs, err := c.probeDNS(ctx, cfg, drv, attachment)
	if err != nil {
		return nil
	}

	conn, err := c.probeTCP(ctx, cfg, drv, attachment, addrs)
	if err != nil {
		return nil
	}

	c.probeTLS(ctx, drv, attachment, credentials, conn)
	return nil
}

func (c *Verify) probeDNS(ctx context.Context, cfg *cli.Config, drv *ui.Driver, attachment api.Attachment) ([]string, error) {
	resolver := cfg.Test.NetResolver
	if resolver == nil {
		resolver = new(net.Resolver)
	}

	domain := attachment.Domains[0]

	mdl := &models.Checker{
		Name: fmt.Sprintf("DNS %s", ui.Emphasize(domain)),
	}
	drv.Activate(ctx, mdl)

	ips, err := resolver.LookupHost(context.Background(), domain)
	if err != nil {
		drv.Send(mdl.Fail(err))
		return nil, err
	}

	drv.Send(mdl.Pass())
	return ips, nil
}

func (c *Verify) probeTCP(ctx context.Context, cfg *cli.Config, drv *ui.Driver, attachment api.Attachment, ips []string) (net.Conn, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	innerDialer := cfg.Test.NetDialer
	if innerDialer == nil {
		innerDialer = new(net.Dialer)
	}

	resolver := cfg.Test.NetResolver
	if resolver == nil {
		resolver = new(net.Resolver)
	}

	var addrs []string
	for _, ip := range ips {
		if strings.Contains(ip, ":") {
			ip = "[" + ip + "]"
		}
		addr := ip + ":" + strconv.Itoa(int(attachment.Port))
		addrs = append(addrs, addr)
	}

	mdl := &models.Checker{
		Name: fmt.Sprintf("TCP [%s]", ui.Emphasize(strings.Join(addrs, ", "))),
	}
	drv.Activate(ctx, mdl)

	connc := make(chan net.Conn)
	for _, addr := range addrs {
		go dialer{
			inner:   innerDialer,
			addr:    addr,
			connc:   connc,
			backoff: 1 * time.Second,
		}.run(ctx)
	}

	select {
	case conn := <-connc:
		drv.Send(mdl.Pass())
		return conn, nil
	case <-ctx.Done():
		drv.Send(mdl.Fail(ctx.Err()))
		return nil, ctx.Err()
	}
}

func (c *Verify) probeTLS(ctx context.Context, drv *ui.Driver, attachment api.Attachment, credentials []api.Credential, conn net.Conn) error {
	domain := attachment.Domains[0]

	mdl := &models.Checker{
		Name: fmt.Sprintf("TLS %s", ui.Emphasize(domain)),
	}
	drv.Activate(ctx, mdl)

	subCAs := x509.NewCertPool()
	for _, cred := range credentials {
		subCAs.AppendCertsFromPEM([]byte(cred.TextualEncoding))
	}

	cfg := &tls.Config{
		ServerName: domain,
		RootCAs:    subCAs,
	}

	if err := tls.Client(conn, cfg).HandshakeContext(ctx); err != nil {
		drv.Send(mdl.Fail(err))
		return err
	}

	drv.Send(mdl.Pass())
	return nil
}

func (c *Verify) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
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
		Prompt: "What is the organization of the service you want to verify?",
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

func (c *Verify) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, serviceAPID string, attachments []api.Attachment) (string, error) {
	if c.RealmAPID == "" && cfg.Lcl.RealmAPID != "" {
		drv.Activate(ctx, &componentmodels.ConfigVia{
			Config:        cfg,
			ConfigFetchFn: func(cfg *cli.Config) any { return cfg.Lcl.RealmAPID },
			Flag:          "--realm",
			Singular:      "realm",
		})
		c.RealmAPID = cfg.Lcl.RealmAPID
	}

	if c.RealmAPID == "" {
		selector := &component.Selector[api.Realm]{
			Prompt: fmt.Sprintf("Which realm of the %s/%s service do you want to verify?", ui.Emphasize(orgAPID), ui.Emphasize(serviceAPID)),
			Flag:   "--realm",

			Fetcher: &component.Fetcher[api.Realm]{
				// TODO: filter GetOrgRealms results to just attachedRealms
				FetchFn: func() ([]api.Realm, error) { return c.anc.GetOrgRealms(ctx, orgAPID) },
			},
		}

		realm, err := selector.Choice(ctx, drv)
		if err != nil {
			return "", err
		}
		c.RealmAPID = realm.Apid
	}

	var attachedRealms []string
	for _, attachment := range attachments {
		attachedRealms = append(attachedRealms, attachment.Relationships.Realm.Apid)
	}

	if slices.Contains(attachedRealms, c.RealmAPID) {
		return c.RealmAPID, nil
	}
	return "", fmt.Errorf("%s service is not attached to the %s/%s realm", serviceAPID, orgAPID, c.RealmAPID)
}

func (c *Verify) serviceAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
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
		Prompt: fmt.Sprintf("Which %s service do you want to verify?", ui.Emphasize(orgAPID)),
		Flag:   "--service",

		Fetcher: &component.Fetcher[api.Service]{
			FetchFn: func() ([]api.Service, error) { return c.anc.GetOrgServices(ctx, orgAPID) },
		},
	}

	service, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return service.Slug, nil
}

type dialer struct {
	inner   cli.Dialer
	addr    string
	connc   chan<- net.Conn
	backoff time.Duration
}

func (d dialer) run(ctx context.Context) {
	for {
		conn, err := d.inner.DialContext(ctx, "tcp", d.addr)
		if err == nil {
			d.connc <- conn
			return
		}

		time.Sleep(d.backoff)
	}
}
