package models

import (
	"fmt"

	"github.com/anchordotdev/cli/ui"
)

var (
	SignOutHeader = ui.Section{
		Name: "SignOutHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Signout from Anchor.dev %s", ui.Whisper("`anchor auth signout`"))),
		},
	}

	SignOutSuccess = ui.Section{
		Name: "SignOutSuccess",
		Model: ui.MessageLines{
			ui.StepDone("Signed out."),
		},
	}
)
