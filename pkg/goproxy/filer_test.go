package goproxy_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/goproxy"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
)

func openTestZip(t *testing.T) *zip.Reader {
	t.Helper()
	zf, err := zip.OpenReader(filepath.Join("testdata", "testify-1.5.1.zip"))
	if err != nil {
		t.Fatalf("test archive open failed: %v", err)
	}

	return &zf.Reader
}

func TestZipFiler_ReadFile(t *testing.T) {
	filer := &goproxy.ZipFiler{
		Reader: openTestZip(t),
		Module: validation.Module{
			Name:    "github.com/stretchr/testify",
			Version: semver.MustParse("v1.5.1"),
		},
	}

	t.Run("read go.mod", func(t *testing.T) {
		content, err := filer.ReadFile("go.mod")
		if assert.NoError(t, err) {
			assert.Equal(t, []byte(`module github.com/stretchr/testify

require (
	github.com/davecgh/go-spew v1.1.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/stretchr/objx v0.1.0
	gopkg.in/yaml.v2 v2.2.2
)

go 1.13
`), content)
		}
	})

	t.Run("read non-existing file", func(t *testing.T) {
		_, err := filer.ReadFile("ohoho")
		assert.Equal(t, &os.PathError{
			Op:   "open",
			Path: "ohoho",
			Err:  os.ErrNotExist,
		}, err)
	})

	t.Run("read directory", func(t *testing.T) {
		_, err := filer.ReadFile("suite")
		assert.Equal(t, &os.PathError{
			Op:   "open",
			Path: "suite",
			Err:  goproxy.ErrNotFile,
		}, err)
	})
}

func TestZipFiler_ReadDir(t *testing.T) {
	filer := &goproxy.ZipFiler{
		Reader: openTestZip(t),
		Module: validation.Module{
			Name:    "github.com/stretchr/testify",
			Version: semver.MustParse("v1.5.1"),
		},
	}

	t.Run("read dir", func(t *testing.T) {
		expectedFiles := []string{
			"doc.go",
			"mock.go",
			"mock_test.go",
		}

		dir, err := filer.ReadDir("mock")
		if assert.NoError(t, err) {
			assert.Len(t, dir, len(expectedFiles), "no files in directory")

			for _, item := range dir {
				assert.False(t, item.IsDir, "'mock' contains only files, found dir %s", item.Name)
				assert.Contains(t, expectedFiles, item.Name)
			}
		}
	})

	t.Run("read file", func(t *testing.T) {
		_, err := filer.ReadDir("go.mod")
		assert.Equal(t, &os.PathError{
			Op:   "readdir",
			Path: "go.mod",
			Err:  goproxy.ErrNotDirectory,
		}, err)
	})
}
