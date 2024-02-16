package models

import "github.com/anchordotdev/cli/truststore"

type preflightStep int

const (
	diff preflightStep = iota
	finishedPreflight

	confirmSync
	noSync
)

type (
	HandleMsg string

	ExpectedCAsMsg []*truststore.CA
	TargetCAsMsg   []*truststore.CA
)
