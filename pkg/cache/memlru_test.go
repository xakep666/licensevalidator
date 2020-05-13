package cache_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xakep666/licensevalidator/pkg/cache"
	"github.com/xakep666/licensevalidator/pkg/validation"
)

func TestMemLRU_ResolveLicense(t *testing.T) {
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

	c, err := cache.NewMemLRU(cache.Direct{
		LicenseResolver: &licenseResolverMock,
	}, 10)
	require.NoError(t, err)

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

func TestMemLRU_ResolveLicense_error_not_cached(t *testing.T) {
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

	c, err := cache.NewMemLRU(cache.Direct{
		LicenseResolver: &licenseResolverMock,
	}, 10)
	require.NoError(t, err)

	_, err = c.ResolveLicense(context.Background(), module)
	assert.True(t, errors.Is(err, expectedErr), "unexpected error", err)

	// 2nd call should be ok
	actualLicense, err := c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}
}

func TestNewMemLRU_ResolveLicense_eviction(t *testing.T) {
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

	module2 := validation.Module{
		Name:    "test-name-2",
		Version: semver.MustParse("v1.0.0"),
	}

	license2 := validation.License{
		Name:   "MIT License",
		SPDXID: "MIT",
	}

	licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(license, nil).Once()
	licenseResolverMock.On("ResolveLicense", mock.Anything, module2).Return(license2, nil).Once()
	licenseResolverMock.On("ResolveLicense", mock.Anything, module).Return(license, nil).Once()

	c, err := cache.NewMemLRU(cache.Direct{
		LicenseResolver: &licenseResolverMock,
	}, 1)
	require.NoError(t, err)

	actualLicense, err := c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}

	// 2nd call should evict first item
	actualLicense, err = c.ResolveLicense(context.Background(), module2)
	if assert.NoError(t, err) {
		assert.Equal(t, license2, actualLicense)
	}

	actualLicense, err = c.ResolveLicense(context.Background(), module)
	if assert.NoError(t, err) {
		assert.Equal(t, license, actualLicense)
	}
}
