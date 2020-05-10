package app

import "time"

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
	CacheTypeRedis  CacheType = "redis"
)

type TracerType string

const (
	ZipkinTracer TracerType = "zipkin"
	JaegerTracer TracerType = "jaeger"
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

	// Trace is optional tracing/telemetry configuration.
	// Tracing will not be enabled if option not provided.
	Trace *Trace
}

// Cache represents cache configuration
type Cache struct {
	// Type is a cache type
	// Available types:
	// * memory
	// * memlru (SizeItems required)
	// * redis
	Type CacheType

	// SizeItems is a maximum items count in memory lru cache
	SizeItems int

	Redis Redis
}

// Redis represents redis configuration
type Redis struct {
	// Addrs is a slice of connection addresses
	// If more than one provided cluster client will be used
	Addrs []string

	// TTL is optional ttl for keys. Keys will not expire when TTL is not set.
	TTL time.Duration

	// PoolSize is a connection pool size. Default value is 10
	PoolSize int

	// DB allows to select db number
	DB int

	// Password is an optional password
	Password string

	// ConnectTimeout is an optional connect timeout
	ConnectTimeout time.Duration

	// ReadTimeout is an optional timeout to receive data
	ReadTimeout time.Duration

	// WriteTimeout is an optional timeout to send data
	WriteTimeout time.Duration
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

// Trace represents opentelemetry configuration
type Trace struct {
	// CollectorAddress is a traces collector address
	CollectorAddress string

	// TracerType is a trace collector type. Available types:
	// * zipkin
	// * jaeger
	TracerType TracerType

	// SampleProbability samples a given fraction of traces. Fractions >= 1 or <= 0 will
	// always sample. If the parent span is sampled, then it's child spans will
	// automatically be sampled
	SampleProbability float64
}
