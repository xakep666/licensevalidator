package athens

import (
	"context"
	"errors"
	"fmt"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

type InternalValidator struct {
	validation.Validator
}

func (v *InternalValidator) Validate(ctx context.Context, req ValidationRequest) error {
	err := v.Validator.Validate(ctx, validation.Module{Name: req.Module, Version: req.Version})
	var (
		blacklistErr     *validation.ErrBlacklistedModule
		deniedLicenseErr *validation.ErrDeniedLicense
	)

	switch {
	case errors.Is(err, nil):
		return nil
	case errors.Is(err, validation.ErrUnknownLicense),
		errors.As(err, &blacklistErr),
		errors.As(err, &deniedLicenseErr):
		return &ErrForbidden{Inner: err}
	default:
		return fmt.Errorf("validator failed: %w", err)
	}
}
