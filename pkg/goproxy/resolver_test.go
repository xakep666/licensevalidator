package goproxy_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/goproxy"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestClient_ResolveLicense(t *testing.T) {
	t.Parallel()
	mockedServerMux := http.NewServeMux()
	mockedServerMux.HandleFunc("/github.com/stretchr/testify/@v/v1.5.1.zip", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("testdata", "testify-1.5.1.zip"))
	})
	mockedServerMux.HandleFunc("/test-invalid-type/@v/v1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_, _ = fmt.Fprint(w, `{"ok": true}`)
	})
	mockedServerMux.HandleFunc("/gone/@v/v1.0.0.zip", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found: /gone/@v/v1.0.0.zip: invalid version: unknown revision v1.0.0", http.StatusGone)
	})
	mockedServerMux.HandleFunc("/github.com/golang/go/@v/list", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mockedServerMux)

	t.Run("detect MIT", func(t *testing.T) {
		client := goproxy.NewClient(zaptest.NewLogger(t), goproxy.ClientParams{
			HTTPClient:          server.Client(),
			BaseURL:             server.URL,
			ConfidenceThreshold: 0.8,
		})

		lic, err := client.ResolveLicense(context.Background(), validation.Module{
			Name:    "github.com/stretchr/testify",
			Version: semver.MustParse("v1.5.1"),
		})
		if assert.NoError(t, err) {
			assert.Equal(t, validation.License{
				Name:   "MIT License",
				SPDXID: "MIT",
			}, lic)
		}
	})

	t.Run("detection fails by threshold", func(t *testing.T) {
		client := goproxy.NewClient(zaptest.NewLogger(t), goproxy.ClientParams{
			HTTPClient:          server.Client(),
			BaseURL:             server.URL,
			ConfidenceThreshold: 0.99,
		})

		_, err := client.ResolveLicense(context.Background(), validation.Module{
			Name:    "github.com/stretchr/testify",
			Version: semver.MustParse("v1.5.1"),
		})
		assert.True(t, errors.Is(err, validation.ErrUnknownLicense), "Expected ErrUnknownLicense, got", err)
	})

	t.Run("invalid content-type", func(t *testing.T) {
		client := goproxy.NewClient(zaptest.NewLogger(t), goproxy.ClientParams{
			HTTPClient:          server.Client(),
			BaseURL:             server.URL,
			ConfidenceThreshold: 0.99,
		})

		_, err := client.ResolveLicense(context.Background(), validation.Module{
			Name:    "test-invalid-type",
			Version: semver.MustParse("v1.0.0"),
		})

		var ctErr goproxy.InvalidContentTypeErr
		if assert.True(t, errors.As(err, &ctErr), "Expected InvalidContentTypeErr, got", err) {
			assert.Equal(t, "application/json", string(ctErr))
		}
	})

	t.Run("not found module", func(t *testing.T) {
		client := goproxy.NewClient(zaptest.NewLogger(t), goproxy.ClientParams{
			HTTPClient:          server.Client(),
			BaseURL:             server.URL,
			ConfidenceThreshold: 0.99,
		})

		_, err := client.ResolveLicense(context.Background(), validation.Module{
			Name:    "gone",
			Version: semver.MustParse("v1.0.0"),
		})
		assert.Equal(t, validation.ErrUnknownLicense, err)
	})

	t.Run("health check", func(t *testing.T) {
		client := goproxy.NewClient(zaptest.NewLogger(t), goproxy.ClientParams{
			HTTPClient:          server.Client(),
			BaseURL:             server.URL,
			ConfidenceThreshold: 0.99,
		})

		err := client.Check(context.Background())
		assert.NoError(t, err)
	})
}
