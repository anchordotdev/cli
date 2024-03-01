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
	Config *cli.Config

	Anc    *api.Session
	Hint   tea.Model
	Source string
}

func (c Client) Perform(ctx context.Context, drv *ui.Driver) (*api.Session, error) {
	var newClientErr, userInfoErr error

	drv.Activate(ctx, &models.Client{})

	if c.Anc == nil {
		c.Anc, newClientErr = api.NewClient(c.Config)
		if newClientErr != nil && !errors.Is(newClientErr, api.ErrSignedOut) {
			return nil, newClientErr
		}
	}

	drv.Send(models.ClientProbed(true))

	if newClientErr == nil {
		_, userInfoErr := c.Anc.UserInfo(ctx)
		if userInfoErr != nil && !errors.Is(userInfoErr, api.ErrSignedOut) {
			return nil, userInfoErr
		}
	}

	drv.Send(models.ClientTested(true))

	if errors.Is(newClientErr, api.ErrSignedOut) || errors.Is(userInfoErr, api.ErrSignedOut) {
		if c.Hint == nil {
			c.Hint = &models.SignInHint{}
		}
		cmd := &SignIn{
			Config: c.Config,
			Hint:   c.Hint,
			Source: c.Source,
		}
		err := cmd.RunTUI(ctx, drv)
		if err != nil {
			return nil, err
		}

		c.Anc, err = api.NewClient(c.Config)
		if err != nil {
			return nil, err
		}
	}

	return c.Anc, nil
}
