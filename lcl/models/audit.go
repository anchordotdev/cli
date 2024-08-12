package models

import (
	"fmt"

	"github.com/anchordotdev/cli/ui"
)

type AuditUnauthenticated bool

var (
	AuditHeader = ui.Section{
		Name: "AuditHeader",
		Model: ui.MessageLines{
			ui.Header(fmt.Sprintf("Audit lcl.host HTTPS Local Development Environment %s", ui.Whisper("`anchor lcl audit`"))),
		},
	}

	AuditHint = ui.Section{
		Name: "AuditHint",
		Model: ui.MessageLines{
			ui.StepHint("We'll compare your local development CA certificates from Anchor and your local trust stores."),
		},
	}
)
