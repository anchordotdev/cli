package service

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
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

		cmd.Flags().StringVar(&cfg.Service.Env.Method, "method", "", "Integration method for environment variables.")
		cmd.Flags().StringVarP(&cfg.Service.Env.Org, "org", "o", "", "Organization to trust.")
		cmd.Flags().StringVarP(&cfg.Service.Env.Realm, "realm", "r", "", "Realm to trust.")
		cmd.Flags().StringVarP(&cfg.Service.Env.Service, "service", "s", "", "Service for ENV.")

		cmd.Hidden = true
	})

	MethodClipboard = "clipboard"
	MethodDotenv    = "dotenv"
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

	if c.ChainAPID == "" {
		c.ChainAPID = "ca"
	}

	if c.OrgAPID == "" {
		c.OrgAPID = cfg.Service.Env.Org
	}
	if c.OrgAPID == "" {
		choicec, err := component.OrgSelector(ctx, drv, c.Anc,
			"Which organization's env do you want to fetch?",
		)
		if err != nil {
			return err
		}
		select {
		case org := <-choicec:
			c.OrgAPID = org.Slug
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if c.RealmAPID == "" {
		c.RealmAPID = cfg.Service.Env.Realm
	}
	if c.RealmAPID == "" {
		choicec, err := component.RealmSelector(ctx, drv, c.Anc, c.OrgAPID,
			fmt.Sprintf("Which %s realm's env do you want to fetch?", ui.Emphasize(c.OrgAPID)),
		)
		if err != nil {
			return err
		}
		select {
		case realm := <-choicec:
			c.RealmAPID = realm.Apid
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if c.ServiceAPID == "" {
		c.ServiceAPID = cfg.Service.Env.Service
	}
	if c.ServiceAPID == "" {
		choicec, err := component.ServiceSelector(ctx, drv, c.Anc, c.OrgAPID,
			fmt.Sprintf("Which %s/%s service do you want to fetch?", ui.Emphasize(c.OrgAPID), ui.Emphasize(c.RealmAPID)),
		)
		if err != nil {
			return err
		}
		select {
		case service := <-choicec:
			c.ServiceAPID = service.Slug
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if c.whoami == "" {
		userInfo, err := c.Anc.UserInfo(ctx)
		if err != nil {
			return err
		}
		c.whoami = userInfo.Whoami
	}

	drv.Activate(ctx, &models.EnvFetch{
		Service: c.ServiceAPID,
	})

	env := make(map[string]string)

	env["ACME_CONTACT"] = c.whoami
	env["ACME_DIRECTORY_URL"] = cfg.AnchorURL + "/" + url.QueryEscape(c.OrgAPID) + "/" + url.QueryEscape(c.RealmAPID) + "/x509/" + c.ChainAPID + "/acme"

	attachments, err := c.Anc.GetServiceAttachments(ctx, c.OrgAPID, c.ServiceAPID)
	if err != nil {
		return err
	}
	var attachment *api.Attachment
	for _, a := range attachments {
		if a.Relationships.Chain.Apid == c.ChainAPID && a.Relationships.Realm.Apid == c.RealmAPID && a.Relationships.SubCa.Apid != nil {
			attachment = &a
			break
		}
	}

	if attachment != nil {
		eab, err := c.Anc.CreateEAB(ctx, c.ChainAPID, c.OrgAPID, c.RealmAPID, c.ServiceAPID, *attachment.Relationships.SubCa.Apid)
		if err != nil {
			return err
		}
		env["ACME_KID"] = eab.Kid
		env["ACME_HMAC_KEY"] = eab.HmacKey
	}

	service, err := c.Anc.GetService(ctx, c.OrgAPID, c.ServiceAPID)
	if err != nil {
		return err
	}
	env["HTTPS_PORT"] = fmt.Sprintf("%d", *service.LocalhostPort)

	if attachment != nil {
		env["SERVER_NAMES"] = strings.Join(attachment.Domains, ",")
	}

	drv.Send(models.EnvFetchedMsg{})

	envMethod := cfg.Service.Env.Method
	if envMethod == "" {
		choicec := make(chan string)
		drv.Activate(ctx, &models.EnvMethod{
			ChoiceCh: choicec,
		})

		select {
		case envMethod = <-choicec:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	switch envMethod {
	case MethodClipboard:
		err := c.clipboardMethod(ctx, drv, env)
		if err != nil {
			return err
		}
	case MethodDotenv:
		err := c.dotenvMethod(ctx, drv, env)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown method: %s. Please choose either `clipboard` or `dotenv`.", envMethod)
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

func (c *Env) clipboardMethod(ctx context.Context, drv *ui.Driver, env map[string]string) error {
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
		Service:     c.ServiceAPID,
	})

	return nil
}

func (c *Env) dotenvMethod(ctx context.Context, drv *ui.Driver, env map[string]string) error {
	var b bytes.Buffer

	var keys []string
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s=\"%s\"\n", key, env[key])
	}
	err := os.WriteFile(".env", b.Bytes(), 0644)
	if err != nil {
		return err
	}

	drv.Activate(ctx, &models.EnvDotenv{
		Service: c.ServiceAPID,
	})

	return nil
}
