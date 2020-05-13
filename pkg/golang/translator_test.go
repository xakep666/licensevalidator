package golang_test

import (
	"context"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/golang"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/stretchr/testify/assert"
)

func TestTranslator_Translate(t *testing.T) {
	t.Parallel()
	type testCase struct {
		Input  string
		Output string
	}

	f := func(tc testCase) {
		t.Helper()

		actual, err := (golang.Translator{}).Translate(context.Background(), validation.Module{
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
		Input:  "go.googlesource.com/text",
		Output: "github.com/golang/text",
	})

	f(testCase{
		Input:  "golang.org/x/net",
		Output: "github.com/golang/net",
	})
}
