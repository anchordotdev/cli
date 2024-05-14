package auth

import (
	"context"
	"errors"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/api"
	"github.com/anchordotdev/cli/auth/models"
	"github.com/anchordotdev/cli/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type Client struct {
	Anc    *api.Session
	Hint   tea.Model
	Source string
}

func (c Client) Perform(ctx context.Context, drv *ui.Driver) (*api.Session, error) {
	cfg := cli.ConfigFromContext(ctx)

	var newClientErr, userInfoErr error

	drv.Activate(ctx, &models.Client{})

	if c.Anc == nil {
		c.Anc, newClientErr = api.NewClient(cfg)
		if newClientErr != nil && !errors.Is(newClientErr, api.ErrSignedOut) {
			return nil, newClientErr
		}
	}

	drv.Send(models.ClientProbed(true))

	if newClientErr == nil {
		_, userInfoErr = c.Anc.UserInfo(ctx)
		if userInfoErr != nil && !errors.Is(userInfoErr, api.ErrSignedOut) {
			return nil, userInfoErr
		}
	}

	drv.Send(models.ClientTested(true))

	if errors.Is(newClientErr, api.ErrSignedOut) || errors.Is(userInfoErr, api.ErrSignedOut) {
		if c.Hint == nil {
			c.Hint = models.SignInHint
		}
		cmd := &SignIn{
			Hint:   c.Hint,
			Source: c.Source,
		}
		ctx = cli.ContextWithConfig(ctx, cfg)
		err := cmd.RunTUI(ctx, drv)
		if err != nil {
			return nil, err
		}

		c.Anc, err = api.NewClient(cfg)
		if err != nil {
			return nil, err
		}
	}

	return c.Anc, nil
}
