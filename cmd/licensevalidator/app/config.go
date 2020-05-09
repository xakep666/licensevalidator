package app

type UnknownLicenseAction string

const (
	UnknownLicenseAllow UnknownLicenseAction = "allow"
	UnknownLicenseWarn  UnknownLicenseAction = "warn"
	UnknownLicenseDeny  UnknownLicenseAction = "deny"
)

type CacheType string

const (
	CacheTypeMemory CacheType = "memory"
	CacheTypeMemLRU CacheType = "memlru"
)

// Config is a top-level app config
type Config struct {
	// Debug is a flag to enable debug logging
	Debug bool

	// Cache is optional cache configuration.
	// Cache will not be used if not present (not recommended).
	Cache *Cache

	Github Github

	GoProxy GoProxy

	// PathOverrides contains set of rules for translation module names
	PathOverrides []OverridePath

	Validation Validation

	Server Server
}

// Cache represents cache configuration
type Cache struct {
	// Type is a cache type
	// Available types:
	// * memory
	// * memlru (SizeItems required)
	Type CacheType

	// SizeItems is a maximum items count in memory lru cache
	SizeItems int
}

// Github contains github client configuration
type Github struct {
	// AccessToken is optional github access token
	// It's needed to access private repos or increase rate-limit
	AccessToken MaskedString
}

// GoProxy contains goproxy client configuration
type GoProxy struct {
	// BaseURL is a goproxy basic url
	// Obviously it must not be athens url which will use this app
	BaseURL MaskedURL
}

// OverridePath is a single override for module path
type OverridePath struct {
	// Match is a regular expression to match module name
	Match string

	// Replace is a replacement string for module name.
	// Regexp capturing group placeholders (i.e $1, $2) may be used here.
	Replace string
}

// ModuleMatcher represents a module matcher configuration
type ModuleMatcher struct {
	// Name is a regular expression for name
	Name string

	// VersionConstraint is a semver version constraint (for syntax see https://github.com/Masterminds/semver/#checking-version-constraints)
	VersionConstraint string `toml:",omitempty"`
}

// License represents a license
type License struct {
	// SPDXID is a spdx license id
	SPDXID string `toml:",omitempty"`

	// Name is a human-readable name
	Name string
}

// Validation contains validator config values
type Validation struct {
	// UnknownLicenseAction specifies what to do if unknown license met.
	// Currently available:
	// * allow - simply allow such module
	// * TODO: warn - allows such module but notifies
	// * deny - fails module validation
	UnknownLicenseAction UnknownLicenseAction

	// ConfidenceThreshold is a lower bound for license matching confidence when it's done by go-license-detector
	ConfidenceThreshold float64

	RuleSet RuleSet
}

// RuleSet defines a validation rule set
type RuleSet struct {
	// WhitelistedModules always passes validation
	WhitelistedModules []ModuleMatcher

	// BlacklistedModules always fails validation
	BlacklistedModules []ModuleMatcher

	// AllowedLicenses contains set of allowed licenses
	// Note that if it's not empty only these licenses will be allowed.
	AllowedLicenses []License

	// DeniedLicenses contains set of denied licenses
	DeniedLicenses []License
}

// Server represents http-server configuration
type Server struct {
	// ListenAddr is a listen address (i.e. ':8080')
	ListenAddr string
	// EnablePprof adds pprof handlers to server at /pprof
	EnablePprof bool
}
