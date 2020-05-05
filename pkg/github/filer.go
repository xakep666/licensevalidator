package github

import (
	"encoding/base64"
	"fmt"

	"github.com/google/go-github/v18/github"
	"gopkg.in/src-d/go-license-detector.v3/licensedb/filer"
)

// filerImpl implements filer.Filer to return the license text directly
// from the github.RepositoryLicense structure.
type filerImpl struct {
	License *github.RepositoryLicense
}

func (f *filerImpl) ReadFile(name string) ([]byte, error) {
	if name != "LICENSE" {
		return nil, fmt.Errorf("unknown file: %s", name)
	}

	return base64.StdEncoding.DecodeString(f.License.GetContent())
}

func (f *filerImpl) ReadDir(dir string) ([]filer.File, error) {
	// We only support root
	if dir != "" {
		return nil, nil
	}

	return []filer.File{{Name: "LICENSE"}}, nil
}

func (f *filerImpl) Close() {}

func (f *filerImpl) PathsAreAlwaysSlash() bool { return true }
