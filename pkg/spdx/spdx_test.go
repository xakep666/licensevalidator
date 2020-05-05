package spdx_test

import (
	"testing"

	"github.com/xakep666/licensevalidator/pkg/spdx"

	"github.com/stretchr/testify/assert"
)

func TestLicenseByID(t *testing.T) {
	t.Run("find MIT", func(t *testing.T) {
		licInfo, ok := spdx.LicenseByID("MIT")
		if assert.True(t, ok, "MIT present in SPDX but not found") {
			assert.Equal(t, spdx.LicenseInfo{
				ID:          "MIT",
				Name:        "MIT License",
				OSIApproved: true,
				SeeAlso:     []string{"https://opensource.org/licenses/MIT"},
			}, licInfo)
		}
	})

	t.Run("non-existent", func(t *testing.T) {
		_, ok := spdx.LicenseByID("asopkpasofaso")
		assert.False(t, ok)
	})
}
