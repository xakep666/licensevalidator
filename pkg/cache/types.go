package cache

import "github.com/xakep666/licensevalidator/pkg/validation"

// Cacher covers all interfaces which calls should be cached
type Cacher interface {
	validation.LicenseResolver
}

type Direct struct {
	validation.LicenseResolver
}
