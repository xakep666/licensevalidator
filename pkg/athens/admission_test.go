package athens_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xakep666/licensevalidator/pkg/athens"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAdmissionHandler(t *testing.T) {
	type testCase struct {
		Name               string
		Request            *http.Request
		ExpectedCode       int
		ExpectedBody       string
		ValidatorMockSetup func(m *athens.ValidatorMock)
	}

	f := func(tc testCase) {
		t.Run(tc.Name, func(t *testing.T) {
			var validatorMock athens.ValidatorMock
			if tc.ValidatorMockSetup != nil {
				tc.ValidatorMockSetup(&validatorMock)
			}

			defer validatorMock.AssertExpectations(t)

			rec := httptest.NewRecorder()

			athens.AdmissionHandler(&validatorMock)(rec, tc.Request)

			assert.Equal(t, tc.ExpectedCode, rec.Code)

			if tc.ExpectedBody != "" {
				// http.Error add newline to end of text
				assert.Equal(t, tc.ExpectedBody, strings.TrimRight(rec.Body.String(), "\n"))
			}
		})
	}

	f(testCase{
		Name:    "module passes validation",
		Request: makeRequest( /*language=json*/ `{"Module":  "test-mod", "Version":  "v1.0.0"}`),
		ValidatorMockSetup: func(m *athens.ValidatorMock) {
			m.On("Validate", mock.Anything, athens.ValidationRequest{
				Module:  "test-mod",
				Version: semver.MustParse("v1.0.0"),
			}).Return(nil).Once()
		},
		ExpectedCode: http.StatusOK,
	})

	f(testCase{
		Name:         "request without version",
		Request:      makeRequest( /*language=json*/ `{"Module":  "test-mod"}`),
		ExpectedCode: http.StatusOK,
	})

	f(testCase{
		Name:         "request without module name",
		Request:      makeRequest( /*language=json*/ `{}`),
		ExpectedCode: http.StatusBadRequest,
		ExpectedBody: "no module name",
	})

	f(testCase{
		Name:    "forbidden module",
		Request: makeRequest( /*language=json*/ `{"Module":  "test-mod", "Version":  "v1.0.0"}`),
		ValidatorMockSetup: func(m *athens.ValidatorMock) {
			m.On("Validate", mock.Anything, athens.ValidationRequest{
				Module:  "test-mod",
				Version: semver.MustParse("v1.0.0"),
			}).Return(&athens.ErrForbidden{
				Inner: fmt.Errorf("module in blacklist"),
			}).Once()
		},
		ExpectedCode: http.StatusForbidden,
		ExpectedBody: "module forbidden: module in blacklist",
	})

	f(testCase{
		Name:    "internal error",
		Request: makeRequest( /*language=json*/ `{"Module":  "test-mod", "Version":  "v1.0.0"}`),
		ValidatorMockSetup: func(m *athens.ValidatorMock) {
			m.On("Validate", mock.Anything, athens.ValidationRequest{
				Module:  "test-mod",
				Version: semver.MustParse("v1.0.0"),
			}).Return(fmt.Errorf("test internal")).Once()
		},
		ExpectedCode: http.StatusInternalServerError,
		ExpectedBody: "test internal",
	})

	f(testCase{
		Name:         "bad method",
		Request:      httptest.NewRequest(http.MethodGet, "/", nil),
		ExpectedCode: http.StatusMethodNotAllowed,
	})

	f(testCase{
		Name: "bad content type",
		Request: func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set("Content-Type", "blabla")
			return req
		}(),
		ExpectedCode: http.StatusNotAcceptable,
	})

	f(testCase{
		Name:         "bad json",
		Request:      makeRequest("{x"),
		ExpectedCode: http.StatusBadRequest,
	})

	f(testCase{
		Name:         "bad version",
		Request:      makeRequest( /*language=json*/ `{"Module":  "test-mod", "Version":  "bla"}`),
		ExpectedCode: http.StatusBadRequest,
	})
}

func TestAdmissionHandler_prevents_misconfiguration(t *testing.T) {
	var validatorMock athens.ValidatorMock
	defer validatorMock.AssertExpectations(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	athens.AdmissionHandler(nil, req.Host, req.RemoteAddr)(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func makeRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}
