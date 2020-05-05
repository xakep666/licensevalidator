package validation

import (
	"fmt"
	"regexp"

	"github.com/Masterminds/semver/v3"
)

// ModuleMatcher defines a module matcher
type ModuleMatcher struct {
	Name    *regexp.Regexp
	Version *semver.Constraints
}

func (mm *ModuleMatcher) String() string {
	if mm.Version == nil {
		return fmt.Sprintf("ModuleMatcher<NameRegex: %s>", mm.Name)
	}

	return fmt.Sprintf("ModuleMatcher<NameRegex: %s, VersionConstraint: %s>", mm.Name, mm.Version)
}

func (mm *ModuleMatcher) Match(m *Module) bool {
	return mm.Name.MatchString(m.Name) && (mm.Version == nil || mm.Version.Check(m.Version))
}

// LicensedModule represents a module with found license
type LicensedModule struct {
	Module
	License License
}

func (lm *LicensedModule) String() string {
	return fmt.Sprintf("LicensedModule<Module: %s, License: %s>", lm.Module, lm.License)
}

// RuleSet represents module validation rule set
type RuleSet struct {
	// WhitelistedModules always gives positive validation result
	WhitelistedModules []ModuleMatcher

	// BlacklistedModules always gives negative validation result
	BlacklistedModules []ModuleMatcher

	// AllowedLicenses contains set of allowed licenses
	// If provided only modules with matched license will be allowed
	AllowedLicenses []License

	// DeniedLicenses contains set of denied licenses
	DeniedLicenses []License
}

// Validate validates provided module against rule set
func (rs *RuleSet) Validate(lm LicensedModule) error {
	for _, wm := range rs.WhitelistedModules {
		if wm.Match(&lm.Module) {
			return nil
		}
	}

	for i, bm := range rs.BlacklistedModules {
		if bm.Match(&lm.Module) {
			return &ErrBlacklistedModule{Module: lm, Matcher: rs.BlacklistedModules[i]}
		}
	}

	for _, al := range rs.AllowedLicenses {
		if al.Equals(&lm.License) {
			return nil
		}
	}

	if len(rs.AllowedLicenses) > 0 {
		return &ErrDeniedLicense{Module: lm}
	}

	for _, dl := range rs.DeniedLicenses {
		if dl.Equals(&lm.License) {
			return &ErrDeniedLicense{Module: lm}
		}
	}

	return nil
}

type ErrBlacklistedModule struct {
	Module  LicensedModule
	Matcher ModuleMatcher
}

func (e *ErrBlacklistedModule) Error() string {
	return fmt.Sprintf("module %s is in blacklist (matched by %s)", e.Module, e.Matcher)
}

type ErrDeniedLicense struct {
	Module LicensedModule
}

func (e *ErrDeniedLicense) Error() string {
	return fmt.Sprintf("module %s has denied license", e.Module)
}
