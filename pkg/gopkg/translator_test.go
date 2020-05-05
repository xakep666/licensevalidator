package gopkg_test

import (
	"context"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/gopkg"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/stretchr/testify/assert"
)

func TestTranslator_Translate(t *testing.T) {
	type testCase struct {
		Input  string
		Output string
	}

	f := func(tc testCase) {
		t.Helper()

		actual, err := (gopkg.Translator{}).Translate(context.Background(), validation.Module{
			Name: tc.Input,
		})
		if assert.NoError(t, err) {
			assert.Equal(t, tc.Output, actual.Name)
		}
	}

	f(testCase{
		Input:  "github.com/foo/bar",
		Output: "github.com/foo/bar",
	})

	f(testCase{
		Input:  "gopkg.in/pkg.v3",
		Output: "github.com/go-pkg/pkg",
	})

	f(testCase{
		Input:  "gopkg.in/yaml.v3",
		Output: "github.com/go-yaml/yaml",
	})

	f(testCase{
		Input:  "gopkg.in/mitchellh/foo.v22",
		Output: "github.com/mitchellh/foo",
	})
}
