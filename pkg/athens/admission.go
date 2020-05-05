package athens

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
)

// AdmissionHandler is a athens admission (validator) web hook handler
// It calls internal validator to check if module can be used.
func AdmissionHandler(validator Validator, forbiddenSources ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		// misconfiguration protection (when target goproxy configured as Athens calling this endpoint)
		for _, fs := range forbiddenSources {
			if fs == r.RemoteAddr || fs == r.Host {
				http.Error(w, "Misconfiguration found, got request from forbidden source (target goproxy)", http.StatusInternalServerError)
				return
			}
		}

		if r.Method != http.MethodPost {
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}

		contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if contentType != "application/json" {
			http.Error(w, "unexpected content-type", http.StatusNotAcceptable)
			return
		}

		var request ValidationRequest

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			http.Error(w, fmt.Sprintf("request parse failed: %s", err), http.StatusBadRequest)
			return
		}

		if request.Module == "" {
			http.Error(w, "no module name", http.StatusBadRequest)
			return
		}

		// no version is ok (i.e. called for version listing)
		if request.Version == nil {
			return
		}

		err = validator.Validate(r.Context(), request)
		var forbiddenErr *ErrForbidden
		switch {
		case errors.Is(err, nil):
			// pass
		case errors.As(err, &forbiddenErr):
			http.Error(w, forbiddenErr.Error(), http.StatusForbidden)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
