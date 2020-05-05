package validation

import (
	"context"
	"errors"
	"fmt"
)

// ChainedTranslator calls all translators until error passing translated module to next translator
type ChainedTranslator struct {
	Translators []Translator
}

func (ct *ChainedTranslator) Translate(ctx context.Context, m Module) (translated Module, err error) {
	for _, translator := range ct.Translators {
		var err error
		m, err = translator.Translate(ctx, m)
		if err != nil {
			return m, err
		}
	}

	return m, nil
}

// ChainedLicenseResolver calls all resolvers until success
// If no success calls happened it returns ErrUnknownLicense
type ChainedLicenseResolver struct {
	LicenseResolvers []LicenseResolver
}

func (crl *ChainedLicenseResolver) ResolveLicense(ctx context.Context, m Module) (License, error) {
	for _, resolver := range crl.LicenseResolvers {
		lic, err := resolver.ResolveLicense(ctx, m)
		switch {
		case errors.Is(err, nil):
			return lic, nil
		case errors.Is(err, ErrUnknownLicense):
			continue
		default:
			return License{}, fmt.Errorf("%w", err)
		}
	}

	return License{}, ErrUnknownLicense
}
