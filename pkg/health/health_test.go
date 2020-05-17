package health_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/xakep666/licensevalidator/pkg/health"
)

func TestHealth(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		args       []health.Option
		statusCode int
		response   health.Response
	}

	f := func(tt testCase) {
		t.Run(tt.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			req, err := http.NewRequest("GET", "http://localhost/health", nil)
			if err != nil {
				t.Errorf("Failed to create request.")
			}
			health.NewHealth(tt.args...).ServeHTTP(res, req)
			if res.Code != tt.statusCode {
				t.Errorf("expected code %d, got %d", tt.statusCode, res.Code)
			}
			var respBody health.Response
			if err := json.NewDecoder(res.Body).Decode(&respBody); err != nil {
				t.Fatal("failed to parse the body")
			}
			if !reflect.DeepEqual(respBody, tt.response) {
				t.Errorf("NewHandlerFunc() = %v, want %v", respBody, tt.response)
			}
		})
	}

	f(testCase{
		name:       "returns 200 status if no errors",
		statusCode: http.StatusOK,
		response: health.Response{
			Status: http.StatusText(http.StatusOK),
		},
	})
	f(testCase{
		name:       "returns 503 status if errors",
		statusCode: http.StatusServiceUnavailable,
		args: []health.Option{
			health.WithChecker("database", health.CheckerFunc(func(ctx context.Context) error {
				return fmt.Errorf("connection to db timed out")
			})),
			health.WithChecker("testService", health.CheckerFunc(func(ctx context.Context) error {
				return fmt.Errorf("connection refused")
			})),
		},
		response: health.Response{
			Status: http.StatusText(http.StatusServiceUnavailable),
			Errors: map[string]string{
				"database":    "connection to db timed out",
				"testService": "connection refused",
			},
		},
	})
	f(testCase{
		name:       "returns 503 status if checkers timeout",
		statusCode: http.StatusServiceUnavailable,
		args: []health.Option{
			health.WithTimeout(1 * time.Millisecond),
			health.WithChecker("database", health.CheckerFunc(func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})),
		},
		response: health.Response{
			Status: http.StatusText(http.StatusServiceUnavailable),
			Errors: map[string]string{
				"database": "max check time exceeded",
			},
		},
	})
	f(testCase{
		name:       "returns 200 status if errors are observable",
		statusCode: http.StatusOK,
		args: []health.Option{
			health.WithObserver("observableService", health.CheckerFunc(func(ctx context.Context) error {
				return fmt.Errorf("i fail but it is okay")
			})),
		},
		response: health.Response{
			Status: http.StatusText(http.StatusOK),
			Errors: map[string]string{
				"observableService": "i fail but it is okay",
			},
		},
	})
	f(testCase{
		name:       "returns 503 status if errors with observable fails",
		statusCode: http.StatusServiceUnavailable,
		args: []health.Option{
			health.WithObserver("database", health.CheckerFunc(func(ctx context.Context) error {
				return fmt.Errorf("connection to db timed out")
			})),
			health.WithChecker("testService", health.CheckerFunc(func(ctx context.Context) error {
				return fmt.Errorf("connection refused")
			})),
		},
		response: health.Response{
			Status: http.StatusText(http.StatusServiceUnavailable),
			Errors: map[string]string{
				"database":    "connection to db timed out",
				"testService": "connection refused",
			},
		},
	})
}
