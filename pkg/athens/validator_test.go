package athens_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/athens"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type InternalValidatorTestSuite struct {
	suite.Suite

	validatorMock *validation.ValidatorMock

	internalValidator *athens.InternalValidator
}

func (s *InternalValidatorTestSuite) TestOk() {
	s.validatorMock.On("Validate", mock.Anything, validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}).Return(nil).Once()

	s.NoError(s.internalValidator.Validate(context.Background(), athens.ValidationRequest{
		Module:  "test",
		Version: semver.MustParse("v1.0.0"),
	}))
}

func (s *InternalValidatorTestSuite) TestGenericError() {
	testErr := fmt.Errorf("test err")

	s.validatorMock.On("Validate", mock.Anything, validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}).Return(testErr).Once()

	err := s.internalValidator.Validate(context.Background(), athens.ValidationRequest{
		Module:  "test",
		Version: semver.MustParse("v1.0.0"),
	})
	s.True(errors.Is(err, testErr), "unexpected error", err)
}

func (s *InternalValidatorTestSuite) TestForbidden() {
	mod := validation.Module{
		Name:    "test",
		Version: semver.MustParse("v1.0.0"),
	}
	req := athens.ValidationRequest{
		Module:  "test",
		Version: semver.MustParse("v1.0.0"),
	}

	var fbErr *athens.ErrForbidden

	s.validatorMock.On("Validate", mock.Anything, mod).Return(validation.ErrUnknownLicense).Once()
	err := s.internalValidator.Validate(context.Background(), req)
	s.True(errors.As(err, &fbErr), "unexpected error", err)
	s.Equal(fbErr.Unwrap(), validation.ErrUnknownLicense)

	blacklistErr := &validation.ErrBlacklistedModule{
		Module:  validation.LicensedModule{Module: mod, License: validation.License{SPDXID: "MIT"}},
		Matcher: validation.ModuleMatcher{Name: regexp.MustCompile(`^test$`)},
	}
	s.validatorMock.On("Validate", mock.Anything, mod).Return(blacklistErr).Once()
	err = s.internalValidator.Validate(context.Background(), req)
	s.True(errors.As(err, &fbErr), "unexpected error", err)
	s.Equal(fbErr.Unwrap(), blacklistErr)

	deniedLicenseErr := &validation.ErrDeniedLicense{
		Module: validation.LicensedModule{Module: mod, License: validation.License{SPDXID: "MIT"}},
	}
	s.validatorMock.On("Validate", mock.Anything, mod).Return(deniedLicenseErr).Once()
	err = s.internalValidator.Validate(context.Background(), req)
	s.True(errors.As(err, &fbErr), "unexpected error", err)
	s.Equal(fbErr.Unwrap(), deniedLicenseErr)
}

func (s *InternalValidatorTestSuite) SetupTest() {
	s.validatorMock = new(validation.ValidatorMock)
	s.internalValidator = &athens.InternalValidator{Validator: s.validatorMock}
}

func (s *InternalValidatorTestSuite) TearDownTest() {
	s.validatorMock.AssertExpectations(s.T())
}

func TestInternalValidator_Suite(t *testing.T) {
	suite.Run(t, new(InternalValidatorTestSuite))
}
