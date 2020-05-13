package preload

import (
	"fmt"

	"gopkg.in/src-d/go-license-detector.v3/licensedb"
	"gopkg.in/src-d/go-license-detector.v3/licensedb/filer"
)

// LicenseDB loads license db. This needed to not slow down first query processing.
// go-license-detector holds license database in asset and loads and parses it.
// This process guarded by sync.Once inside library.
func LicenseDB() {
	_, _ = licensedb.Detect(preloadFiler{})
}

type preloadFiler struct{}

func (preloadFiler) ReadFile(path string) (content []byte, err error) {
	if path != "LICENSE" {
		return nil, fmt.Errorf("unknown file: %s", path)
	}

	return []byte("SOME TEXT"), nil
}

func (preloadFiler) ReadDir(path string) ([]filer.File, error) {
	if path != "" {
		return nil, nil
	}

	return []filer.File{{Name: "LICENSE"}}, nil
}

func (preloadFiler) Close() {}

func (preloadFiler) PathsAreAlwaysSlash() bool { return true }
