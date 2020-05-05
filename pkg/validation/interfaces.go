package validation

import (
	"context"
	"fmt"
)

var ErrUnknownLicense = fmt.Errorf("unknown license")

type Translator interface {
	// Translate attempts to remap module name
	// Example case rsc.io/pdf -> github.com/rsc/pdf
	// Implementation should return original Module if translation wasn't made
	Translate(ctx context.Context, m Module) (translated Module, err error)
}

type LicenseResolver interface {
	// ResolveLicense resolves license for module
	// It should return ErrUnknownLicense if module contains unknown license (not found in db-s)
	ResolveLicense(ctx context.Context, m Module) (License, error)
}

type UnknownLicenseNotifier interface {
	// NotifyUnknownLicense triggered if unknown license found and UnknownLicenseWarn is set
	NotifyUnknownLicense(ctx context.Context, m Module) error
}
