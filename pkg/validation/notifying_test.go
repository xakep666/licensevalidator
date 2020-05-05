package validation_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
)

type NotifyingValidatorTestSuite struct {
	suite.Suite

	validatorMock *validation.ValidatorMock
	notifierMock  *validation.UnknownLicenseNotifierMock
}

func (s *NotifyingValidatorTestSuite) TestSuccess() {
	module := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}

	s.validatorMock.On("Validate", mock.Anything, module).Return(nil).Once()

	s.NoError(validation.NewNotifyingValidator(zaptest.NewLogger(s.T()), validation.NotifyingValidatorParams{
		Validator:              s.validatorMock,
		UnknownLicenseAction:   validation.UnknownLicenseDeny,
		UnknownLicenseNotifier: s.notifierMock,
	}).Validate(context.Background(), module))
}

func (s *NotifyingValidatorTestSuite) TestGenericError() {
	module := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}

	testErr := fmt.Errorf("test err")
	s.validatorMock.On("Validate", mock.Anything, module).Return(testErr).Once()

	err := validation.NewNotifyingValidator(zaptest.NewLogger(s.T()), validation.NotifyingValidatorParams{
		Validator:              s.validatorMock,
		UnknownLicenseAction:   validation.UnknownLicenseDeny,
		UnknownLicenseNotifier: s.notifierMock,
	}).Validate(context.Background(), module)
	s.True(errors.Is(err, testErr), "unexpected error", err)
}

func (s *NotifyingValidatorTestSuite) TestUnknownLicenseAllow() {
	module := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}

	s.validatorMock.On("Validate", mock.Anything, module).Return(validation.ErrUnknownLicense).Once()

	s.NoError(validation.NewNotifyingValidator(zaptest.NewLogger(s.T()), validation.NotifyingValidatorParams{
		Validator:              s.validatorMock,
		UnknownLicenseAction:   validation.UnknownLicenseAllow,
		UnknownLicenseNotifier: s.notifierMock,
	}).Validate(context.Background(), module))
}

func (s *NotifyingValidatorTestSuite) TestUnknownLicenseNotify() {
	module := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}

	s.validatorMock.On("Validate", mock.Anything, module).Return(validation.ErrUnknownLicense).Once()
	s.notifierMock.On("NotifyUnknownLicense", mock.Anything, module).Return(nil).Once()

	s.NoError(validation.NewNotifyingValidator(zaptest.NewLogger(s.T()), validation.NotifyingValidatorParams{
		Validator:              s.validatorMock,
		UnknownLicenseAction:   validation.UnknownLicenseWarn,
		UnknownLicenseNotifier: s.notifierMock,
	}).Validate(context.Background(), module))
}

func (s *NotifyingValidatorTestSuite) TestUnknownLicenseDeny() {
	module := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}

	s.validatorMock.On("Validate", mock.Anything, module).Return(validation.ErrUnknownLicense).Once()

	err := validation.NewNotifyingValidator(zaptest.NewLogger(s.T()), validation.NotifyingValidatorParams{
		Validator:              s.validatorMock,
		UnknownLicenseAction:   validation.UnknownLicenseDeny,
		UnknownLicenseNotifier: s.notifierMock,
	}).Validate(context.Background(), module)
	s.True(errors.Is(err, validation.ErrUnknownLicense), "unexpected error", err)
}

func (s *NotifyingValidatorTestSuite) SetupTest() {
	s.validatorMock = new(validation.ValidatorMock)
	s.notifierMock = new(validation.UnknownLicenseNotifierMock)
}

func (s *NotifyingValidatorTestSuite) TearDownTest() {
	s.notifierMock.AssertExpectations(s.T())
	s.validatorMock.AssertExpectations(s.T())
}

func TestNotifyingValidator_Suite(t *testing.T) {
	suite.Run(t, new(NotifyingValidatorTestSuite))
}
