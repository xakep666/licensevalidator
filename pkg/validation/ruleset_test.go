package validation_test

import (
	"regexp"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func MustParseConstraint(constraint string) *semver.Constraints {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		panic(err)
	}

	return c
}

func TestRuleSet_Validate(t *testing.T) {
	t.Parallel()
	type testCase struct {
		Name          string
		Module        validation.LicensedModule
		RuleSet       validation.RuleSet
		ExpectedError error
	}

	f := func(tc testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.ExpectedError == nil {
				assert.NoError(t, tc.RuleSet.Validate(tc.Module))
			} else {
				assert.Equal(t, tc.ExpectedError, tc.RuleSet.Validate(tc.Module))
			}
		})

	}

	f(testCase{
		Name: "empty set is ok",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
	})

	f(testCase{
		Name: "whitelist matched",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
		RuleSet: validation.RuleSet{
			WhitelistedModules: []validation.ModuleMatcher{
				{Name: regexp.MustCompile("github.com/stretchr/testify"), Version: MustParseConstraint(">=1.0.0")},
			},
		},
	})

	f(testCase{
		Name: "blacklist denied",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
		RuleSet: validation.RuleSet{
			BlacklistedModules: []validation.ModuleMatcher{
				{Name: regexp.MustCompile("github.com/stretchr/testify"), Version: MustParseConstraint(">=1.0.0")},
			},
		},
		ExpectedError: &validation.ErrBlacklistedModule{
			Module: validation.LicensedModule{
				Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
				License: validation.License{Name: "MIT License", SPDXID: "MIT"},
			},
			Matcher: validation.ModuleMatcher{
				Name:    regexp.MustCompile("github.com/stretchr/testify"),
				Version: MustParseConstraint(">=1.0.0"),
			},
		},
	})

	f(testCase{
		Name: "license not in whitelist",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
		RuleSet: validation.RuleSet{
			AllowedLicenses: []validation.License{
				{Name: "GNU GPL v3", SPDXID: "GPL3"},
			},
		},
		ExpectedError: &validation.ErrDeniedLicense{
			Module: validation.LicensedModule{
				Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
				License: validation.License{Name: "MIT License", SPDXID: "MIT"},
			},
		},
	})

	f(testCase{
		Name: "license in blacklist",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
		RuleSet: validation.RuleSet{
			DeniedLicenses: []validation.License{
				{Name: "MIT License", SPDXID: "MIT"},
			},
		},
		ExpectedError: &validation.ErrDeniedLicense{
			Module: validation.LicensedModule{
				Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
				License: validation.License{Name: "MIT License", SPDXID: "MIT"},
			},
		},
	})

	f(testCase{
		Name: "license in whitelist",
		Module: validation.LicensedModule{
			Module:  validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
			License: validation.License{Name: "MIT License", SPDXID: "MIT"},
		},
		RuleSet: validation.RuleSet{
			AllowedLicenses: []validation.License{
				{Name: "MIT License", SPDXID: "MIT"},
			},
		},
	})
}

func TestModuleMatcher_Match(t *testing.T) {
	t.Parallel()
	type testCase struct {
		Name          string
		Module        validation.Module
		Matcher       validation.ModuleMatcher
		ExpectedMatch bool
	}

	f := func(tc testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			assert.Equal(t, tc.ExpectedMatch, tc.Matcher.Match(&tc.Module))
		})

	}

	f(testCase{
		Name:          "match only by name",
		Module:        validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
		Matcher:       validation.ModuleMatcher{Name: regexp.MustCompile("github.com/stretchr/testify")},
		ExpectedMatch: true,
	})

	f(testCase{
		Name:          "match only by name fails",
		Module:        validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
		Matcher:       validation.ModuleMatcher{Name: regexp.MustCompile("haha")},
		ExpectedMatch: false,
	})

	f(testCase{
		Name:          "match by name and version",
		Module:        validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
		Matcher:       validation.ModuleMatcher{Name: regexp.MustCompile("github.com/stretchr/testify"), Version: MustParseConstraint(">=1.0.0")},
		ExpectedMatch: true,
	})

	f(testCase{
		Name:          "match by name and version fails",
		Module:        validation.Module{Name: "github.com/stretchr/testify", Version: semver.MustParse("v1.2.3")},
		Matcher:       validation.ModuleMatcher{Name: regexp.MustCompile("github.com/stretchr/testify"), Version: MustParseConstraint(">=2.0.0")},
		ExpectedMatch: false,
	})
}
