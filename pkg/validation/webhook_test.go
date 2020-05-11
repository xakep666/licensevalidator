package validation_test

import (
	"context"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

func TestLicenseResolverMock_ResolveLicense(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	t.Run("webhook success", func(t *testing.T) {
		mux.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			b, err := ioutil.ReadAll(r.Body)
			require.NoError(t, err)

			assert.JSONEq(t, `{"module": {"name": "test-module", "version": "1.0.0"}}`, string(b))
		})

		err := validation.NewWebhookNotifier(zaptest.NewLogger(t), validation.WebhookNotifierParams{
			Client:       server.Client(),
			Address:      server.URL + "/hook",
			BodyTemplate: template.Must(template.New("").Parse(`{"module": {"name": "{{.Module.Name}}", "version": "{{.Module.Version}}"}}`)),
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		}).NotifyUnknownLicense(context.Background(), validation.Module{
			Name:    "test-module",
			Version: semver.MustParse("v1.0.0"),
		})

		assert.NoError(t, err)
	})

	t.Run("webhook with bad server status", func(t *testing.T) {
		mux.HandleFunc("/badhook", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		err := validation.NewWebhookNotifier(zaptest.NewLogger(t), validation.WebhookNotifierParams{
			Client:       server.Client(),
			Address:      server.URL + "/badhook",
			BodyTemplate: template.Must(template.New("").Parse("test")),
		}).NotifyUnknownLicense(context.Background(), validation.Module{
			Name:    "test-module",
			Version: semver.MustParse("v1.0.0"),
		})

		assert.Error(t, err)
	})

	t.Run("empty request body", func(t *testing.T) {
		mux.HandleFunc("/emptybody", func(w http.ResponseWriter, r *http.Request) {})

		err := validation.NewWebhookNotifier(zaptest.NewLogger(t), validation.WebhookNotifierParams{
			Client:       server.Client(),
			Address:      server.URL + "/emptybody",
			BodyTemplate: template.Must(template.New("").Parse("")),
		}).NotifyUnknownLicense(context.Background(), validation.Module{
			Name:    "test-module",
			Version: semver.MustParse("v1.0.0"),
		})

		assert.NoError(t, err)
	})
}
