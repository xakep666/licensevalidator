package validation_test

import (
	"context"
	"errors"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type ValidatorTestSuite struct {
	suite.Suite

	TranslatorMock      *validation.TranslatorMock
	LicenseResolverMock *validation.LicenseResolverMock
}

func (s *ValidatorTestSuite) Test_all_ok() {
	module := validation.Module{
		Name:    "test-module",
		Version: semver.MustParse("v1.2.3"),
	}

	s.TranslatorMock.On("Translate", mock.Anything, module).Return(module, nil).Once()
	s.LicenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}, nil).Once()

	s.NoError(validation.NewRuleSetValidator(zaptest.NewLogger(s.T()), validation.RuleSetValidatorParams{
		Translator:      s.TranslatorMock,
		LicenseResolver: s.LicenseResolverMock,
		RuleSet:         validation.RuleSet{},
	}).Validate(context.Background(), module))
}

func (s *ValidatorTestSuite) Test_with_translation() {
	module := validation.Module{
		Name:    "test-module",
		Version: semver.MustParse("v1.2.3"),
	}

	translated := validation.Module{
		Name:    "test-module-tr",
		Version: semver.MustParse("v1.2.3"),
	}

	s.TranslatorMock.On("Translate", mock.Anything, module).Return(translated, nil).Once()
	s.LicenseResolverMock.On("ResolveLicense", mock.Anything, translated).Return(validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}, nil).Once()

	s.NoError(validation.NewRuleSetValidator(zaptest.NewLogger(s.T()), validation.RuleSetValidatorParams{
		Translator:      s.TranslatorMock,
		LicenseResolver: s.LicenseResolverMock,
		RuleSet:         validation.RuleSet{},
	}).Validate(context.Background(), module))
}

func (s *ValidatorTestSuite) Test_with_translation_resolve_by_original() {
	module := validation.Module{
		Name:    "test-module",
		Version: semver.MustParse("v1.2.3"),
	}

	translated := validation.Module{
		Name:    "test-module-tr",
		Version: semver.MustParse("v1.2.3"),
	}

	s.TranslatorMock.On("Translate", mock.Anything, module).Return(translated, nil).Once()
	s.LicenseResolverMock.On("ResolveLicense", mock.Anything, translated).
		Return(validation.License{}, validation.ErrUnknownLicense).Once()
	s.LicenseResolverMock.On("ResolveLicense", mock.Anything, module).
		Return(validation.License{
			Name:   "MIT License",
			SPDXID: "MIT",
		}, nil).
		Once()

	s.NoError(validation.NewRuleSetValidator(zaptest.NewLogger(s.T()), validation.RuleSetValidatorParams{
		Translator:      s.TranslatorMock,
		LicenseResolver: s.LicenseResolverMock,
		RuleSet:         validation.RuleSet{},
	}).Validate(context.Background(), module))
}

func (s *ValidatorTestSuite) Test_unknown_license() {
	module := validation.Module{
		Name:    "test-module",
		Version: semver.MustParse("v1.2.3"),
	}

	s.TranslatorMock.On("Translate", mock.Anything, module).Return(module, nil).Once()
	s.LicenseResolverMock.On("ResolveLicense", mock.Anything, module).
		Return(validation.License{}, validation.ErrUnknownLicense).Once()

	err := validation.NewRuleSetValidator(zaptest.NewLogger(s.T()), validation.RuleSetValidatorParams{
		Translator:      s.TranslatorMock,
		LicenseResolver: s.LicenseResolverMock,
		RuleSet:         validation.RuleSet{},
	}).Validate(context.Background(), module)
	s.True(errors.Is(err, validation.ErrUnknownLicense), "unexpected error", err)
}

func (s *ValidatorTestSuite) SetupSuite() {
	s.TranslatorMock = new(validation.TranslatorMock)
	s.LicenseResolverMock = new(validation.LicenseResolverMock)
}

func (s *ValidatorTestSuite) TearDownSuite() {
	s.TranslatorMock.AssertExpectations(s.T())
	s.LicenseResolverMock.AssertExpectations(s.T())
}

func TestValidator_Suite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ValidatorTestSuite))
}
