# Project license validator for Athens proxy
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/xakep666/licensevalidator)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xakep666/licensevalidator)](https://goreportcard.com/report/github.com/xakep666/licensevalidator)
[![codecov](https://codecov.io/gh/xakep666/licensevalidator/branch/master/graph/badge.svg)](https://codecov.io/gh/xakep666/licensevalidator)
[![Docker Pulls](https://img.shields.io/docker/pulls/xakep666/licensevalidator.svg)](https://img.shields.io/docker/pulls/xakep666/licensevalidator.svg)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fxakep666%2Flicensevalidator.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fxakep666%2Flicensevalidator?ref=badge_shield)

## Why?
Direct or transitive dependencies may have license like [AGPL-3.0](https://spdx.org/licenses/AGPL-3.0.html) that enforces user to open-source product.
So this project should help to deal with such situations.
Related articles:
* https://www.agwa.name/blog/post/always_review_your_dependencies

## How this project can help me?
It's a web-server that handles [Athens proxy server](https://gomods.io/) validation webhook requests.
This hook called each time when user tries to download module through Athens. This project receives request and performs validation according to settings. If validation fails Athens receives HTTP 403 (Forbidden) status and doesn't allow module downloading.

## Features
* Flexible rule system:
    * Blacklist modules by name or version constraint (i.e. forbid modules with version less than 1.0.0)
    * Whitelist modules by name or version (i.e. always allow modules from your internal repos)
    * Allow only modules licensed by configured licenses
    * Deny modules licensed by configured licenses
    * License can be defined by [SPDX License List](https://spdx.org/licenses/) id or human-readable name.
* Configurable behaviour for modules with non-determined license:
    * Allow such modules
    * Deny such modules
* Dealing with vanity servers (servers needed for decoupling module name from repository like gopkg.in). Project supports gopkg.in, golang.org/x and go.googlesource.com out of the box. Other rewrite rules can be added through config
* Multiple sources of license detection:
    * Github for modules hosted on it. Has fallback to [go-license-detector](godoc.org/gopkg.in/src-d/go-license-detector.v3)
    * Detection using module zip from proxy.golang.org with [go-license-detector](godoc.org/gopkg.in/src-d/go-license-detector.v3) without downloading whole zip


[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fxakep666%2Flicensevalidator.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fxakep666%2Flicensevalidator?ref=badge_large)

## Running
* Direct install:
    * `go install github.com/xakep666/licensevalidator/cmd/licensevalidator`
    * Generate config example and tune it `licensevalidator sample-config > config.toml`
    * Start a service `licensevalidator -c config.toml`
* Use pre-build [docker image](https://hub.docker.com/repository/docker/xakep666/licensevalidator/general).
 Configuration can be bind-mount to `/etc/licensevalidator.toml`

You can manually check if module allowed to use by running making HTTP POST to `/athens/admission` with body
```json
{
    "Module": "github.com/stretchr/testify",
    "Version": "v1.5.1"
}
```
Note that header `Content-Type: application/json` is required.

## Configuration
Example config can be received by running `licensevalidator sample-config`
Here it is with some comments (more comments in [config.go](./cmd/licensevalidator/app/config.go)).
```toml
# enable debug logging
Debug = true

# Cache for some heavy operations (currently license resolution operation).
# It's not recommended to disable it.
[Cache]
  Type = "memory"

[Github]
  # Provide github access token to decrease rate-limit
  AccessToken = "test-github-token"

[GoProxy]
  # URL of goproxy server that will be used for license detection
  # Obviously it should not be address of Athens server which calls this app.
  BaseURL = "https://proxy.golang.org"

# Path overrides for vanity servers
# This example holds rule for modules published by Uber
[[PathOverrides]]
  Match = "^go.uber.org/(.*)$"
  Replace = "github.com/uber-go/$1"

# Web server settings
[Server]
  ListenAddr = ":8080"
  EnablePprof = true # adds pprof handlers at /pprof

[Validation]
  # Some ways of license detection doesn't produce 100% accurate result.
  # This parameter holds lower-bound threshold of license matching confidence.
  ConfidenceThreshold = 0.8

  # How to deal with unknown licenses: allow or deny
  UnknownLicenseAction = "allow"

  [Validation.RuleSet]

    # Allowed licenses list. If not empty only modules with provided licenses can be used.
    [[Validation.RuleSet.AllowedLicenses]]
      SPDXID = "MIT"

    # If module will be matched by these rules it will be blocked anyway.
    [[Validation.RuleSet.BlacklistedModules]]
      Name = "rsc.io/pdf"
      # for constraint syntax see https://github.com/Masterminds/semver/#checking-version-constraints
      VersionConstraint = "<1.0.0"

    # Module with denied licenses will be blocked.
    [[Validation.RuleSet.DeniedLicenses]]
      SPDXID = "AGPL-3.0"

    # Modules matching whitelist always allowed.
    [[Validation.RuleSet.WhitelistedModules]]
      Name = "^gitlab.mycorp.com/.*"

    [[Validation.RuleSet.WhitelistedModules]]
      Name = "github.com/user/repo"
      VersionConstraint = ">=1.0.0"
```

Athens proxy should be configured properly by setting `ATHENS_PROXY_VALIDATOR` environment variable or `ValidatorHook` parameter in config to `<base-url of app>/athens/admission`

## Caveats
* Regexp-based [go-license-detector](godoc.org/gopkg.in/src-d/go-license-detector.v3) is slow, very slow. Simple license detection (only single file with license text) takes approx 2s on MacBook Pro (15-inch, 2017)

## Running tests
This project contains integration tests that uses [testcontainers-go](https://github.com/testcontainers/testcontainers-go).
They can be skipped using `-short` flag. Correct running requires working Docker.
For running tests inside container be sure that management is available inside container
i.e docker socket bind-mounted into container `-v /var/run/docker.socket:/var/run/docker.socket` and network mode is `host`.

## Plans
- [ ] Notifying about unknown license
- [ ] Better instrumentation: prometheus and opentelemetry
- [ ] More cache variants: In-memory LRU, Redis
- [ ] Live example to try project without installation
- [ ] Be k8s-friendly: add proper liveness and readiness checks and helm chart
- [ ] Improve performance for methods involving go-license-detector

and more...