package main

import (
	"io"
	"os"

	"github.com/xakep666/licensevalidator/cmd/licensevalidator/app"

	"github.com/pelletier/go-toml"
	"github.com/urfave/cli/v2"
)

var ConfigSample = app.Config{
	Debug: true,
	Cache: &app.Cache{
		Type: app.CacheTypeMemory,
	},
	GoProxy: app.GoProxy{
		BaseURL: "https://proxy.golang.org",
	},
	PathOverrides: []app.OverridePath{
		{
			Match:   `^go.uber.org/(.*)$`,
			Replace: `github.com/uber-go/$1`,
		},
	},
	Validation: app.Validation{
		UnknownLicenseAction: app.UnknownLicenseAllow,
		ConfidenceThreshold:  0.8,
		RuleSet: app.RuleSet{
			WhitelistedModules: []app.ModuleMatcher{
				{Name: "^gitlab.mycorp.com/.*"},
				{Name: "github.com/user/repo", VersionConstraint: ">=1.0.0"},
			},
			BlacklistedModules: []app.ModuleMatcher{
				{Name: "rsc.io/pdf", VersionConstraint: "<1.0.0"},
			},
			AllowedLicenses: []app.License{
				{SPDXID: "MIT"},
			},
			DeniedLicenses: []app.License{
				{SPDXID: "AGPL-3.0"},
			},
		},
	},
	Server: app.Server{
		ListenAddr:  ":8080",
		EnablePprof: true,
	},
	HealthServer: &app.Server{
		ListenAddr: ":8081",
	},
}

var configSampleOut io.Writer = os.Stdout // for mocking

func ConfigSampleCommand() *cli.Command {
	return &cli.Command{
		Name:        "sample-config",
		Description: "Prints sample config file to stdout",
		Action: func(ctx *cli.Context) error {
			return toml.NewEncoder(configSampleOut).Encode(ConfigSample)
		},
	}
}
