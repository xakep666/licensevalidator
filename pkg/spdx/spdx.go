// This package contains full license list from SPDX
// It may be regenerated (wget and mule required)
package spdx

import (
	"encoding/json"
	"sync"
)

//go:generate wget -O licenses.json https://spdx.org/licenses/licenses.json
//go:generate mule -p $GOPACKAGE licenses.json

type LicenseList struct {
	// Version is the raw version string of the license list.
	Version string `json:"licenseListVersion"`

	// Licenses is the list of known licenses.
	Licenses []LicenseInfo `json:"licenses"`
}

// LicenseInfo is a single software license.
//
// Basic descriptions are documented in the fields below.
// For a full description of the fields, see the official SPDX specification here:
// https://github.com/spdx/license-list-data/blob/master/accessingLicenses.md
type LicenseInfo struct {
	ID          string   `json:"licenseId"`
	Name        string   `json:"name"`
	Text        string   `json:"licenseText"`
	Deprecated  bool     `json:"isDeprecatedLicenseId"`
	OSIApproved bool     `json:"isOsiApproved"`
	SeeAlso     []string `json:"seeAlso"`
}

var (
	licenseIDIndex   map[string]LicenseInfo
	licenseIndexOnce sync.Once
)

func initIndexes() {
	licenseIndexOnce.Do(func() {
		listBytes, err := licensesResource()
		if err != nil {
			panic(err)
		}

		var list LicenseList
		err = json.Unmarshal(listBytes, &list)
		if err != nil {
			panic(err)
		}

		licenseIDIndex = make(map[string]LicenseInfo)

		for _, item := range list.Licenses {
			licenseIDIndex[item.ID] = item
		}
	})
}

func LicenseByID(id string) (LicenseInfo, bool) {
	initIndexes()

	info, ok := licenseIDIndex[id]
	return info, ok
}
