package testflags

import (
	"flag"

	"github.com/spf13/pflag"
)

func init() {
	pflag.CommandLine.String("api-lockfile", "tmp/apitest.lock", "rails server lockfile path")
	if err := pflag.CommandLine.MarkHidden("api-lockfile"); err != nil {
		panic(err)
	}

	pflag.CommandLine.String("oapi-config", "config/openapi.yml", "openapi spec file path")
	if err := pflag.CommandLine.MarkHidden("oapi-config"); err != nil {
		panic(err)
	}

	pflag.CommandLine.Bool("update", false, "update .golden files")
	if err := pflag.CommandLine.MarkHidden("update"); err != nil {
		panic(err)
	}

	pflag.CommandLine.Bool("prism-proxy", false, "run prism in proxy mode")
	if err := pflag.CommandLine.MarkHidden("prism-proxy"); err != nil {
		panic(err)
	}

	pflag.CommandLine.Bool("prism-verbose", false, "run prism in verbose mode")
	if err := pflag.CommandLine.MarkHidden("prism-verbose"); err != nil {
		panic(err)
	}

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}
