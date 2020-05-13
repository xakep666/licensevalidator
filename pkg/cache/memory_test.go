package cache_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/cache"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMemoryCache_ResolveLicense(t *testing.T) {
	t.Parallel()
	var licenseResolverMock validation.LicenseResolverMock
	defer licenseResolverMock.AssertExpectations(t)

	module := validation.Module{
		Name:    "test-name",
		Version: semver.MustParse("v1.0.0"),
	}

	license := validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}

	licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(license, nil).Once()

	c := cache.MemoryCache{Backed: cache.Direct{
		LicenseResolver: &licenseResolverMock,
	}}

	actualLicense, err := c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}

	// 2nd call should be in cache
	actualLicense, err = c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}
}

func TestMemoryCache_ResolveLicense_error_not_cached(t *testing.T) {
	t.Parallel()
	var licenseResolverMock validation.LicenseResolverMock
	defer licenseResolverMock.AssertExpectations(t)

	module := validation.Module{
		Name:    "test-name",
		Version: semver.MustParse("v1.0.0"),
	}

	license := validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}

	expectedErr := fmt.Errorf("test-err")
	licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(validation.License{}, expectedErr).Once()
	licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(license, nil).Once()

	c := cache.MemoryCache{Backed: cache.Direct{
		LicenseResolver: &licenseResolverMock,
	}}

	_, err := c.ResolveLicense(context.Background(), module)
	assert.True(t, errors.Is(err, expectedErr), "unexpected error", err)

	// 2nd call should be ok
	actualLicense, err := c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}
}
