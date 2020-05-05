package validation

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
)

type RuleSetValidatorParams struct {
	Translator      Translator
	LicenseResolver LicenseResolver
	RuleSet         RuleSet
}

type RuleSetValidator struct {
	RuleSetValidatorParams

	log *zap.Logger
}

func NewRuleSetValidator(logger *zap.Logger, validatorParams RuleSetValidatorParams) *RuleSetValidator {
	return &RuleSetValidator{
		RuleSetValidatorParams: validatorParams,
		log:                    logger.With(zap.String("component", "ruleset_validator")),
	}
}

func (v *RuleSetValidator) Validate(ctx context.Context, m Module) error {
	l := v.log.With(zap.Stringer("module", &m))
	l.Info("Validating module")

	// firstly we try resolve license by translated module to minimize resolving using slow methods

	translated, err := v.Translator.Translate(ctx, m)
	if err != nil {
		return fmt.Errorf("translation failed: %w", err)
	}

	l = l.With(zap.Stringer("translated", &translated))
	l.Debug("Translated module")

	lic, err := v.LicenseResolver.ResolveLicense(ctx, translated)
	switch {
	case errors.Is(err, nil):
		// pass
	case errors.Is(err, ErrUnknownLicense):
		if m.Name == translated.Name {
			l.Warn("Module has unknown license and translation didn't happen")
			return ErrUnknownLicense
		}

		l.Info("Translated module license not resolved. Trying to resolve license for original module")
		lic, err = v.tryOriginalModule(ctx, m)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("license resolution failed: %w", err)
	}

	err = v.RuleSet.Validate(LicensedModule{Module: m, License: lic})
	if err != nil {
		return fmt.Errorf("rule set validation failed: %w", err)
	}

	return nil
}

func (v *RuleSetValidator) tryOriginalModule(ctx context.Context, original Module) (License, error) {
	lic, err := v.LicenseResolver.ResolveLicense(ctx, original)
	if err != nil {
		return License{}, fmt.Errorf("failed to resolve license by original module: %w", err)
	}

	return lic, nil
}
