package github_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xakep666/licensevalidator/pkg/github"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/Masterminds/semver/v3"
	gh "github.com/google/go-github/v18/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	mitJSON = /*language=json*/ `{
  "name": "LICENSE",
  "path": "LICENSE",
  "sha": "4b0421cf9ee47908beae4b4648babb75b09ee028",
  "size": 1103,
  "url": "https://api.github.com/repos/stretchr/testify/contents/LICENSE?ref=master",
  "html_url": "https://github.com/stretchr/testify/blob/master/LICENSE",
  "git_url": "https://api.github.com/repos/stretchr/testify/git/blobs/4b0421cf9ee47908beae4b4648babb75b09ee028",
  "download_url": "https://raw.githubusercontent.com/stretchr/testify/master/LICENSE",
  "type": "file",
  "content": "TUlUIExpY2Vuc2UKCkNvcHlyaWdodCAoYykgMjAxMi0yMDIwIE1hdCBSeWVy\nLCBUeWxlciBCdW5uZWxsIGFuZCBjb250cmlidXRvcnMuCgpQZXJtaXNzaW9u\nIGlzIGhlcmVieSBncmFudGVkLCBmcmVlIG9mIGNoYXJnZSwgdG8gYW55IHBl\ncnNvbiBvYnRhaW5pbmcgYSBjb3B5Cm9mIHRoaXMgc29mdHdhcmUgYW5kIGFz\nc29jaWF0ZWQgZG9jdW1lbnRhdGlvbiBmaWxlcyAodGhlICJTb2Z0d2FyZSIp\nLCB0byBkZWFsCmluIHRoZSBTb2Z0d2FyZSB3aXRob3V0IHJlc3RyaWN0aW9u\nLCBpbmNsdWRpbmcgd2l0aG91dCBsaW1pdGF0aW9uIHRoZSByaWdodHMKdG8g\ndXNlLCBjb3B5LCBtb2RpZnksIG1lcmdlLCBwdWJsaXNoLCBkaXN0cmlidXRl\nLCBzdWJsaWNlbnNlLCBhbmQvb3Igc2VsbApjb3BpZXMgb2YgdGhlIFNvZnR3\nYXJlLCBhbmQgdG8gcGVybWl0IHBlcnNvbnMgdG8gd2hvbSB0aGUgU29mdHdh\ncmUgaXMKZnVybmlzaGVkIHRvIGRvIHNvLCBzdWJqZWN0IHRvIHRoZSBmb2xs\nb3dpbmcgY29uZGl0aW9uczoKClRoZSBhYm92ZSBjb3B5cmlnaHQgbm90aWNl\nIGFuZCB0aGlzIHBlcm1pc3Npb24gbm90aWNlIHNoYWxsIGJlIGluY2x1ZGVk\nIGluIGFsbApjb3BpZXMgb3Igc3Vic3RhbnRpYWwgcG9ydGlvbnMgb2YgdGhl\nIFNvZnR3YXJlLgoKVEhFIFNPRlRXQVJFIElTIFBST1ZJREVEICJBUyBJUyIs\nIFdJVEhPVVQgV0FSUkFOVFkgT0YgQU5ZIEtJTkQsIEVYUFJFU1MgT1IKSU1Q\nTElFRCwgSU5DTFVESU5HIEJVVCBOT1QgTElNSVRFRCBUTyBUSEUgV0FSUkFO\nVElFUyBPRiBNRVJDSEFOVEFCSUxJVFksCkZJVE5FU1MgRk9SIEEgUEFSVElD\nVUxBUiBQVVJQT1NFIEFORCBOT05JTkZSSU5HRU1FTlQuIElOIE5PIEVWRU5U\nIFNIQUxMIFRIRQpBVVRIT1JTIE9SIENPUFlSSUdIVCBIT0xERVJTIEJFIExJ\nQUJMRSBGT1IgQU5ZIENMQUlNLCBEQU1BR0VTIE9SIE9USEVSCkxJQUJJTElU\nWSwgV0hFVEhFUiBJTiBBTiBBQ1RJT04gT0YgQ09OVFJBQ1QsIFRPUlQgT1Ig\nT1RIRVJXSVNFLCBBUklTSU5HIEZST00sCk9VVCBPRiBPUiBJTiBDT05ORUNU\nSU9OIFdJVEggVEhFIFNPRlRXQVJFIE9SIFRIRSBVU0UgT1IgT1RIRVIgREVB\nTElOR1MgSU4gVEhFClNPRlRXQVJFLgo=\n",
  "encoding": "base64",
  "_links": {
    "self": "https://api.github.com/repos/stretchr/testify/contents/LICENSE?ref=master",
    "git": "https://api.github.com/repos/stretchr/testify/git/blobs/4b0421cf9ee47908beae4b4648babb75b09ee028",
    "html": "https://github.com/stretchr/testify/blob/master/LICENSE"
  },
  "license": {
    "key": "mit",
    "name": "MIT License",
    "spdx_id": "MIT",
    "url": "https://api.github.com/licenses/mit",
    "node_id": "MDc6TGljZW5zZTEz"
  }
}`
	otherJSONWithMIT = /*language=json*/ `{
  "name": "LICENSE",
  "path": "LICENSE",
  "sha": "4b0421cf9ee47908beae4b4648babb75b09ee028",
  "size": 1103,
  "url": "https://api.github.com/repos/stretchr/testify/contents/LICENSE?ref=master",
  "html_url": "https://github.com/stretchr/testify/blob/master/LICENSE",
  "git_url": "https://api.github.com/repos/stretchr/testify/git/blobs/4b0421cf9ee47908beae4b4648babb75b09ee028",
  "download_url": "https://raw.githubusercontent.com/stretchr/testify/master/LICENSE",
  "type": "file",
  "content": "TUlUIExpY2Vuc2UKCkNvcHlyaWdodCAoYykgMjAxMi0yMDIwIE1hdCBSeWVy\nLCBUeWxlciBCdW5uZWxsIGFuZCBjb250cmlidXRvcnMuCgpQZXJtaXNzaW9u\nIGlzIGhlcmVieSBncmFudGVkLCBmcmVlIG9mIGNoYXJnZSwgdG8gYW55IHBl\ncnNvbiBvYnRhaW5pbmcgYSBjb3B5Cm9mIHRoaXMgc29mdHdhcmUgYW5kIGFz\nc29jaWF0ZWQgZG9jdW1lbnRhdGlvbiBmaWxlcyAodGhlICJTb2Z0d2FyZSIp\nLCB0byBkZWFsCmluIHRoZSBTb2Z0d2FyZSB3aXRob3V0IHJlc3RyaWN0aW9u\nLCBpbmNsdWRpbmcgd2l0aG91dCBsaW1pdGF0aW9uIHRoZSByaWdodHMKdG8g\ndXNlLCBjb3B5LCBtb2RpZnksIG1lcmdlLCBwdWJsaXNoLCBkaXN0cmlidXRl\nLCBzdWJsaWNlbnNlLCBhbmQvb3Igc2VsbApjb3BpZXMgb2YgdGhlIFNvZnR3\nYXJlLCBhbmQgdG8gcGVybWl0IHBlcnNvbnMgdG8gd2hvbSB0aGUgU29mdHdh\ncmUgaXMKZnVybmlzaGVkIHRvIGRvIHNvLCBzdWJqZWN0IHRvIHRoZSBmb2xs\nb3dpbmcgY29uZGl0aW9uczoKClRoZSBhYm92ZSBjb3B5cmlnaHQgbm90aWNl\nIGFuZCB0aGlzIHBlcm1pc3Npb24gbm90aWNlIHNoYWxsIGJlIGluY2x1ZGVk\nIGluIGFsbApjb3BpZXMgb3Igc3Vic3RhbnRpYWwgcG9ydGlvbnMgb2YgdGhl\nIFNvZnR3YXJlLgoKVEhFIFNPRlRXQVJFIElTIFBST1ZJREVEICJBUyBJUyIs\nIFdJVEhPVVQgV0FSUkFOVFkgT0YgQU5ZIEtJTkQsIEVYUFJFU1MgT1IKSU1Q\nTElFRCwgSU5DTFVESU5HIEJVVCBOT1QgTElNSVRFRCBUTyBUSEUgV0FSUkFO\nVElFUyBPRiBNRVJDSEFOVEFCSUxJVFksCkZJVE5FU1MgRk9SIEEgUEFSVElD\nVUxBUiBQVVJQT1NFIEFORCBOT05JTkZSSU5HRU1FTlQuIElOIE5PIEVWRU5U\nIFNIQUxMIFRIRQpBVVRIT1JTIE9SIENPUFlSSUdIVCBIT0xERVJTIEJFIExJ\nQUJMRSBGT1IgQU5ZIENMQUlNLCBEQU1BR0VTIE9SIE9USEVSCkxJQUJJTElU\nWSwgV0hFVEhFUiBJTiBBTiBBQ1RJT04gT0YgQ09OVFJBQ1QsIFRPUlQgT1Ig\nT1RIRVJXSVNFLCBBUklTSU5HIEZST00sCk9VVCBPRiBPUiBJTiBDT05ORUNU\nSU9OIFdJVEggVEhFIFNPRlRXQVJFIE9SIFRIRSBVU0UgT1IgT1RIRVIgREVB\nTElOR1MgSU4gVEhFClNPRlRXQVJFLgo=\n",
  "encoding": "base64",
  "_links": {
    "self": "https://api.github.com/repos/stretchr/testify/contents/LICENSE?ref=master",
    "git": "https://api.github.com/repos/stretchr/testify/git/blobs/4b0421cf9ee47908beae4b4648babb75b09ee028",
    "html": "https://github.com/stretchr/testify/blob/master/LICENSE"
  },
  "license": {
    "key": "other",
    "name": "MIT License",
    "spdx_id": "MIT",
    "url": "https://api.github.com/licenses/mit",
    "node_id": "MDc6TGljZW5zZTEz"
  }
}`
)

func TestClient_ResolveLicense(t *testing.T) {
	mockedServerMux := http.NewServeMux()
	mockedServerMux.HandleFunc("/repos/test/mit/license", func(w http.ResponseWriter, r *http.Request) {
		serveJSON(w, http.StatusOK, json.RawMessage(mitJSON))
	})
	mockedServerMux.HandleFunc("/repos/test/other-mit-file/license", func(w http.ResponseWriter, r *http.Request) {
		serveJSON(w, http.StatusOK, json.RawMessage(otherJSONWithMIT))
	})
	mockedServerMux.HandleFunc("/repos/test/rate-limit-mit/license", func() http.HandlerFunc {
		calls := uint32(0)
		// simulate rate-limit each 2nd call
		return func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddUint32(&calls, 1)&1 > 0 {
				w.Header().Set("X-RateLimit-Limit", "60")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10))

				serveJSON(w, http.StatusForbidden, gh.ErrorResponse{
					Message:          "API rate limit exceeded for xxx.xxx.xxx.xxx. (But here's the good news: Authenticated requests get a higher rate limit. Check out the documentation for more details.)",
					DocumentationURL: "https://developer.github.com/v3/#rate-limiting",
				})
				return
			}

			serveJSON(w, http.StatusOK, json.RawMessage(mitJSON))
		}
	}())

	mockedServer := httptest.NewServer(mockedServerMux)

	ghClient, err := gh.NewEnterpriseClient(mockedServer.URL, mockedServer.URL, mockedServer.Client())
	require.NoError(t, err)

	t.Run("resolve license ok", func(t *testing.T) {
		lic, err := github.NewClient(zaptest.NewLogger(t), github.ClientParams{
			Client: ghClient,
		}).ResolveLicense(context.Background(), validation.Module{
			Name:    "github.com/test/mit",
			Version: semver.MustParse("v1.0.0"),
		})

		if assert.NoError(t, err) {
			assert.Equal(t, validation.License{
				Name:   "MIT License",
				SPDXID: "MIT",
			}, lic)
		}
	})

	t.Run("resolve license OK with rate limit", func(t *testing.T) {
		client := github.NewClient(zaptest.NewLogger(t), github.ClientParams{
			Client: ghClient,
		})
		module := validation.Module{
			Name:    "github.com/test/rate-limit-mit",
			Version: semver.MustParse("v1.0.0"),
		}

		start := time.Now()
		lic, err := client.ResolveLicense(context.Background(), module)
		dur := time.Since(start)

		if assert.NoError(t, err) {
			assert.Equal(t, validation.License{
				Name:   "MIT License",
				SPDXID: "MIT",
			}, lic)
		}

		assert.True(t, dur > time.Second, "is rate limit working?")
	})

	t.Run("resolve license fallback", func(t *testing.T) {
		lic, err := github.NewClient(zaptest.NewLogger(t), github.ClientParams{
			Client:                      ghClient,
			FallbackConfidenceThreshold: 0.8,
		}).ResolveLicense(context.Background(), validation.Module{
			Name:    "github.com/test/other-mit-file",
			Version: semver.MustParse("v1.0.0"),
		})

		if assert.NoError(t, err) {
			assert.Equal(t, validation.License{
				Name:   "MIT License",
				SPDXID: "MIT",
			}, lic)
		}
	})
}

func serveJSON(w http.ResponseWriter, code int, object interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(object)
}
