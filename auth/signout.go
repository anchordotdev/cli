package auth

import (
	"context"

	"github.com/muesli/termenv"

	"github.com/anchordotdev/cli"
	"github.com/anchordotdev/cli/keyring"
)

type SignOut struct {
	Config *cli.Config
}

func (s SignOut) TUI() cli.TUI {
	return cli.TUI{
		Run: s.run,
	}
}

func (s *SignOut) run(ctx context.Context, tty termenv.File) error {
	kr := keyring.Keyring{Config: s.Config}
	return kr.Delete(keyring.APIToken)
}
