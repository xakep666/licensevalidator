package override_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/override"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestTranslator_Translate(t *testing.T) {
	type testCase struct {
		Overrides []override.TranslateOverride
		Input     string
		Output    string
		Error     error
	}

	f := func(tc testCase) {
		t.Helper()

		tr := override.NewTranslator(zaptest.NewLogger(t), tc.Overrides)
		actual, err := tr.Translate(context.Background(), validation.Module{
			Name:    tc.Input,
			Version: semver.MustParse("v1.0.0"),
		})

		if tc.Error != nil {
			assert.EqualError(t, err, tc.Error.Error())
			return
		}

		if assert.NoError(t, err) {
			assert.Equal(t, tc.Output, actual.Name)
		}
	}

	f(testCase{
		Overrides: nil,
		Input:     "github.com/foo/bar",
		Output:    "github.com/foo/bar",
	})

	f(testCase{
		Overrides: []override.TranslateOverride{
			{
				Match:   regexp.MustCompile("^gopkg.in/pkg.v3$"),
				Replace: "github.com/go-pkg/pkg",
			},
		},
		Input:  "gopkg.in/pkg.v3",
		Output: "github.com/go-pkg/pkg",
	})

	f(testCase{
		Overrides: []override.TranslateOverride{
			{
				Match:   regexp.MustCompile(`^gopkg\.in/([^/]+)/([^/]+)\.(v\d+)`),
				Replace: `github.com/$1/$2`,
			},
		},
		Input:  "gopkg.in/mitchellh/foo.v22",
		Output: "github.com/mitchellh/foo",
	})

	f(testCase{
		Overrides: []override.TranslateOverride{
			{
				Match:   regexp.MustCompile(`^gop\.in/([^/]+)/([^/]+)\.`),
				Replace: `github.com/$1/$2`,
			},
		},
		Input:  "gopkg.in/mitchellh/foo.v22",
		Output: "gopkg.in/mitchellh/foo.v22",
	})
}
