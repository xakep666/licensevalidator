// Package validation contains logic to perform project validation against provided rules
// This package contains only interfaces for module translation (to deal with vanity servers) and license resolution
// and it should not include such logic.
package validation

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// Module represents go module
type Module struct {
	Name    string
	Version *semver.Version
}

func (m *Module) String() string {
	return fmt.Sprintf("Module<name: %s, version: %s>", m.Name, m.Version)
}

type UnknownLicenseAction int

const (
	// UnknownLicenseAllow allows modules with non-determined license
	UnknownLicenseAllow UnknownLicenseAction = iota

	// UnknownLicenseWarn acts as UnknownLicenseAllow but explicitly notifies about it
	UnknownLicenseWarn

	// UnknownLicenseDeny fails validation for modules with non-determined license
	UnknownLicenseDeny
)

type License struct {
	// Name is a human-readable name
	Name string

	// SPDXID is a SPDX license id
	SPDXID string
}

func (l *License) Equals(other *License) bool {
	if other == nil || l == nil {
		return l == other
	}

	// if spdx id is available compare using it
	if l.SPDXID != "" && other.SPDXID != "" {
		return l.SPDXID == other.SPDXID
	}

	// otherwise compare human-readable names
	return l.Name == other.Name
}

func (l *License) String() string {
	if l == nil || (*l == License{}) {
		return "<unknown license>"
	}

	return fmt.Sprintf("License<Name: %s, SPDX: %s>", l.Name, l.SPDXID)
}

type Validator interface {
	Validate(ctx context.Context, m Module) error
}
