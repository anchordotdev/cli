package service

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth"
	"github.com/anchordotdev/cli/component"
	"github.com/anchordotdev/cli/service/models"
	"github.com/anchordotdev/cli/ui"
	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
)

var (
	CmdServiceEnv = cli.NewCmd[Env](CmdService, "env", func(cmd *cobra.Command) {
		cfg := cli.ConfigFromCmd(cmd)

		cmd.Flags().StringVar(&cfg.Service.EnvOutput, "env-output", cli.Defaults.Service.EnvOutput, "Integration method for environment variables.")
		cmd.Flags().StringVarP(&cfg.Org.APID, "org", "o", cli.Defaults.Org.APID, "Organization to trust.")
		cmd.Flags().StringVarP(&cfg.Realm.APID, "realm", "r", cli.Defaults.Realm.APID, "Realm to trust.")
		cmd.Flags().StringVarP(&cfg.Service.APID, "service", "s", cli.Defaults.Service.APID, "Service for ENV.")

		// alias

		cmd.Flags().StringVar(&cfg.Service.EnvOutput, "method", cli.Defaults.Service.EnvOutput, "Integration method for environment variables.")
		_ = cmd.Flags().MarkDeprecated("method", "Please use `--env-output` instead.")
	})

	MethodExport  = "export"
	MethodDotenv  = "dotenv"
	MethodDisplay = "display"
)

type Env struct {
	Anc *api.Session

	ChainAPID, OrgAPID, RealmAPID, ServiceAPID string

	whoami string
}

func (c Env) UI() cli.UI {
	return cli.UI{
		RunTUI: c.run,
	}
}

func (c *Env) run(ctx context.Context, drv *ui.Driver) error {
	var err error
	clientCmd := &auth.Client{
		Anc: c.Anc,
	}
	c.Anc, err = clientCmd.Perform(ctx, drv)
	if err != nil {
		return err
	}

	drv.Activate(ctx, models.ServiceEnvHeader)
	drv.Activate(ctx, models.ServiceEnvHint)

	err = c.Perform(ctx, drv)
	if err != nil {
		return err
	}

	return nil
}

func (c *Env) Perform(ctx context.Context, drv *ui.Driver) error {
	cfg := cli.ConfigFromContext(ctx)

	chainAPID := c.chainAPID()
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

	if c.whoami == "" {
		userInfo, err := c.Anc.UserInfo(ctx)
		if err != nil {
			return err
		}
		c.whoami = userInfo.Whoami
	}

	drv.Activate(ctx, &models.EnvFetch{
		Service: serviceAPID,
	})

	env := make(map[string]string)

	env["ACME_CONTACT"] = c.whoami
	env["ACME_DIRECTORY_URL"] = cfg.AcmeURL(orgAPID, realmAPID, chainAPID)

	attachments, err := c.Anc.GetServiceAttachments(ctx, orgAPID, serviceAPID)
	if err != nil {
		return err
	}
	var attachment *api.Attachment
	for _, a := range attachments {
		if a.Relationships.Chain.Apid == chainAPID && a.Relationships.Realm.Apid == realmAPID && a.Relationships.SubCa.Apid != nil {
			attachment = &a
			break
		}
	}

	if attachment != nil {
		eab, err := c.Anc.CreateEAB(ctx, chainAPID, orgAPID, realmAPID, serviceAPID, *attachment.Relationships.SubCa.Apid)
		if err != nil {
			return err
		}
		env["ACME_KID"] = eab.Kid
		env["ACME_HMAC_KEY"] = eab.HmacKey
	}

	service, err := c.Anc.GetService(ctx, orgAPID, serviceAPID)
	if err != nil {
		return err
	}
	env["HTTPS_PORT"] = fmt.Sprintf("%d", *service.LocalhostPort)

	if attachment != nil {
		env["SERVER_NAMES"] = strings.Join(attachment.Domains, ",")
	}

	drv.Send(models.EnvFetchedMsg{})

	envOutput := cfg.Service.EnvOutput
	if envOutput == "" {
		choicec := make(chan string)
		drv.Activate(ctx, &models.EnvMethod{
			ChoiceCh: choicec,
		})

		select {
		case envOutput = <-choicec:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	switch envOutput {
	case MethodExport:
		err := c.exportMethod(ctx, drv, env, serviceAPID)
		if err != nil {
			return err
		}
	case MethodDotenv:
		err := c.dotenvMethod(ctx, drv, env, serviceAPID)
		if err != nil {
			return err
		}
	case MethodDisplay:
		err := c.displayMethod(ctx, drv, env, serviceAPID)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown method: %s. Please choose either `export` or `dotenv`.", envOutput)
	}

	var lclDomain string
	for _, d := range attachment.Domains {
		if strings.HasSuffix(d, ".lcl.host") {
			lclDomain = d
			break
		}
	}
	lclUrl := fmt.Sprintf("https://%s:%d", lclDomain, *service.LocalhostPort)

	drv.Activate(ctx, &models.EnvNextSteps{
		LclUrl: lclUrl,
	})

	return nil
}

func (c *Env) exportMethod(ctx context.Context, drv *ui.Driver, env map[string]string, serviceAPID string) error {
	var b strings.Builder

	var keys []string
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "export %s=\"%s\"\n", key, env[key])
	}
	clipboardErr := clipboard.WriteAll(b.String())

	drv.Activate(ctx, &models.EnvClipboard{
		InClipboard: (clipboardErr == nil),
		Service:     serviceAPID,
	})

	return nil
}

func (c *Env) dotenvMethod(ctx context.Context, drv *ui.Driver, env map[string]string, serviceAPID string) error {
	var b bytes.Buffer

	var keys []string
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=\"%s\"\n", key, env[key])
	}
	clipboardErr := clipboard.WriteAll(b.String())

	drv.Activate(ctx, &models.EnvDotenv{
		InClipboard: (clipboardErr == nil),
		Service:     serviceAPID,
	})

	return nil
}

func (c *Env) displayMethod(ctx context.Context, drv *ui.Driver, env map[string]string, serviceAPID string) error {
	var b strings.Builder

	var keys []string
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "export %s=\"%s\"\n", key, env[key])
	}

	drv.Activate(ctx, &models.EnvDisplay{
		EnvString: b.String(),
		Service:   serviceAPID,
	})

	return nil
}

func (c *Env) chainAPID() string {
	if c.ChainAPID != "" {
		return c.ChainAPID
	}

	return "ca"
}

func (c *Env) orgAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver) (string, error) {
	if c.OrgAPID != "" {
		return c.OrgAPID, nil
	}
	if cfg.Org.APID != "" {
		return cfg.Org.APID, nil
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

func (c *Env) realmAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID string) (string, error) {
	if c.RealmAPID != "" {
		return c.RealmAPID, nil
	}
	if cfg.Realm.APID != "" {
		return cfg.Realm.APID, nil
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

func (c *Env) serviceAPID(ctx context.Context, cfg *cli.Config, drv *ui.Driver, orgAPID, realmAPID string) (string, error) {
	if c.ServiceAPID != "" {
		return c.ServiceAPID, nil
	}
	if cfg.Service.APID != "" {
		return cfg.Service.APID, nil
	}

	selector := &component.Selector[api.Service]{
		Prompt: fmt.Sprintf("Which %s/%s service do you want to fetch?", ui.Emphasize(orgAPID), ui.Emphasize(realmAPID)),
		Flag:   "--service",

		Fetcher: &component.Fetcher[api.Service]{
			FetchFn: func() ([]api.Service, error) { return c.Anc.GetOrgServices(ctx, orgAPID, api.NonDiagnosticServices) },
		},
	}

	service, err := selector.Choice(ctx, drv)
	if err != nil {
		return "", err
	}
	return service.Slug, nil
}
