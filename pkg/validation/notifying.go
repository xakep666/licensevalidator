package validation

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

type NotifyingValidatorParams struct {
	Validator              Validator
	UnknownLicenseAction   UnknownLicenseAction
	UnknownLicenseNotifier UnknownLicenseNotifier
}

// NotifyingValidator is a wrapper for Validator interface which performs notifications if requested by user
type NotifyingValidator struct {
	NotifyingValidatorParams

	log *zap.Logger
}

func NewNotifyingValidator(log *zap.Logger, params NotifyingValidatorParams) *NotifyingValidator {
	return &NotifyingValidator{
		NotifyingValidatorParams: params,
		log:                      log.With(zap.String("component", "notifying_validator")),
	}
}

func (v *NotifyingValidator) Validate(ctx context.Context, m Module) error {
	err := v.Validator.Validate(ctx, m)
	switch {
	case errors.Is(err, nil):
		return nil
	case errors.Is(err, ErrUnknownLicense):
		return v.onUnknownLicense(ctx, m)
	default:
		return fmt.Errorf("%w", err)
	}
}

func (v *NotifyingValidator) onUnknownLicense(ctx context.Context, m Module) error {
	l := v.log.With(zap.Stringer("module", &m))

	switch v.UnknownLicenseAction {
	case UnknownLicenseAllow:
		l.Debug("Allowing unknown license")
		return nil
	case UnknownLicenseWarn:
		l.Warn("Notifying about unknown license")
		if err := v.UnknownLicenseNotifier.NotifyUnknownLicense(ctx, m); err != nil {
			l.Error("Notifying about unknown license failed", zap.Error(err))
		}
		return nil
	case UnknownLicenseDeny:
		l.Warn("Denying unknown license")
		return ErrUnknownLicense
	}

	return fmt.Errorf("unknown license action: %v", v.UnknownLicenseAction)
}
