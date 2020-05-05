package validation_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestChainedTranslator_Translate(t *testing.T) {
	var (
		tr1, tr2, tr3 validation.TranslatorMock
	)
	defer tr1.AssertExpectations(t)
	defer tr2.AssertExpectations(t)
	defer tr3.AssertExpectations(t)

	tr1.On("Translate", mock.Anything, validation.Module{
		Name:    "name",
		Version: semver.MustParse("v1.0.0"),
	}).Return(validation.Module{
		Name:    "tr1-name",
		Version: semver.MustParse("v1.0.0"),
	}, nil).Once()

	tr2.On("Translate", mock.Anything, validation.Module{
		Name:    "tr1-name",
		Version: semver.MustParse("v1.0.0"),
	}).Return(validation.Module{
		Name:    "tr2-name",
		Version: semver.MustParse("v1.0.0"),
	}, nil).Once()

	tr3.On("Translate", mock.Anything, validation.Module{
		Name:    "tr2-name",
		Version: semver.MustParse("v1.0.0"),
	}).Return(validation.Module{
		Name:    "tr2-name",
		Version: semver.MustParse("v1.0.0"),
	}, nil).Once()

	actual, err := (&validation.ChainedTranslator{
		Translators: []validation.Translator{&tr1, &tr2, &tr3},
	}).Translate(context.Background(), validation.Module{
		Name:    "name",
		Version: semver.MustParse("v1.0.0"),
	})
	if assert.NoError(t, err) {
		assert.Equal(t, validation.Module{
			Name:    "tr2-name",
			Version: semver.MustParse("v1.0.0"),
		}, actual)
	}
}

func TestChainedTranslator_Translate_error(t *testing.T) {
	var (
		tr1, tr2 validation.TranslatorMock
	)
	defer tr1.AssertExpectations(t)
	defer tr2.AssertExpectations(t)

	mod := validation.Module{
		Name:    "name",
		Version: semver.MustParse("v1.0.0"),
	}
	testErr := fmt.Errorf("test-err")

	tr1.On("Translate", mock.Anything, mod).Return(validation.Module{}, testErr).Once()

	_, err := (&validation.ChainedTranslator{
		Translators: []validation.Translator{&tr1, &tr2},
	}).Translate(context.Background(), mod)

	assert.True(t, errors.Is(err, testErr), "unexpected error", err)
}

func TestChainedLicenseResolver_ResolveLicense(t *testing.T) {
	var (
		r1, r2 validation.LicenseResolverMock
	)
	defer r1.AssertExpectations(t)
	defer r2.AssertExpectations(t)

	mod := validation.Module{
		Name:    "name",
		Version: semver.MustParse("v1.0.0"),
	}
	lic := validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}

	r1.On("ResolveLicense", mock.Anything, mod).Return(validation.License{}, validation.ErrUnknownLicense).Once()
	r2.On("ResolveLicense", mock.Anything, mod).Return(lic, nil).Once()

	actualLic, err := (&validation.ChainedLicenseResolver{
		LicenseResolvers: []validation.LicenseResolver{&r1, &r2},
	}).ResolveLicense(context.Background(), mod)
	if assert.NoError(t, err) {
		assert.Equal(t, lic, actualLic)
	}
}

func TestChainedLicenseResolver_ResolveLicense_error(t *testing.T) {
	var (
		r1, r2 validation.LicenseResolverMock
	)
	defer r1.AssertExpectations(t)
	defer r2.AssertExpectations(t)

	mod := validation.Module{
		Name:    "name",
		Version: semver.MustParse("v1.0.0"),
	}
	testErr := fmt.Errorf("test-err")

	r1.On("ResolveLicense", mock.Anything, mod).Return(validation.License{}, testErr).Once()

	_, err := (&validation.ChainedLicenseResolver{
		LicenseResolvers: []validation.LicenseResolver{&r1, &r2},
	}).ResolveLicense(context.Background(), mod)

	assert.True(t, errors.Is(err, testErr), "unexpected error", err)
}
