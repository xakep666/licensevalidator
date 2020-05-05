package validation

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type ValidatorMock struct {
	mock.Mock
}

func (m *ValidatorMock) Validate(ctx context.Context, module Module) error {
	return m.Called(ctx, module).Error(0)
}

type TranslatorMock struct {
	mock.Mock
}

func (m *TranslatorMock) Translate(ctx context.Context, module Module) (translated Module, err error) {
	args := m.Called(ctx, module)
	return args.Get(0).(Module), args.Error(1)
}

type LicenseResolverMock struct {
	mock.Mock
}

func (m *LicenseResolverMock) ResolveLicense(ctx context.Context, module Module) (License, error) {
	args := m.Called(ctx, module)
	return args.Get(0).(License), args.Error(1)
}

type UnknownLicenseNotifierMock struct {
	mock.Mock
}

func (m *UnknownLicenseNotifierMock) NotifyUnknownLicense(ctx context.Context, module Module) error {
	return m.Called(ctx, module).Error(0)
}
