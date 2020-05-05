package goproxy

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"

	"github.com/xakep666/licensevalidator/pkg/spdx"
	"github.com/xakep666/licensevalidator/pkg/validation"

	bufra "github.com/avvmoto/buf-readerat"
	"github.com/xakep666/httpreaderat/v2"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-license-detector.v3/licensedb"
	"gopkg.in/src-d/go-license-detector.v3/licensedb/api"
)

type InvalidContentTypeErr string

func (e InvalidContentTypeErr) Error() string {
	return fmt.Sprintf("invalid content type: %s", string(e))
}

type ClientParams struct {
	HTTPClient *http.Client

	// BaseURL is a proxy base url (i.e. https://proxy.golang.org)
	BaseURL string

	// StoreMemLimit is a limit for in-memory zip storage when server not supports http range requests
	// By default it's 1MiB
	StoreMemLimit int64

	// StoreFileLimit is a limit for file-based zip storage when server not supports http range requests
	// By default it's 1GiB
	StoreFileLimit int64

	// ConfidenceThreshold is a lower bound threshold of license matching confidence
	ConfidenceThreshold float64
}

type Client struct {
	ClientParams

	log *zap.Logger
}

func NewClient(logger *zap.Logger, params ClientParams) *Client {
	return &Client{
		ClientParams: params,
		log:          logger.With(zap.String("component", "goproxy_client")),
	}
}

// ResolveLicense attempts to resolve license using project zip file.
// Content-Type must be application/zip otherwise InvalidContentTypeErr error returned.
// It uses http range requests to not fully download file when server supports it.
func (c *Client) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	l := c.log.With(zap.Stringer("module", &m))
	moduleZIPPath := fmt.Sprintf("%s/%s/@v/%s.zip", c.BaseURL, m.Name, m.Version.Original())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, moduleZIPPath, nil)
	if err != nil {
		return validation.License{}, fmt.Errorf("construct request failed: %w", err)
	}

	store := c.makeStore()
	defer store.Close()

	var codeErr *httpreaderat.ErrUnexpectedResponseCode
	rd, err := httpreaderat.New(c.HTTPClient, req, store)
	switch {
	case errors.Is(err, nil):
		// pass
	case errors.As(err, &codeErr):
		l.Error("unexpected status code", zap.Error(err))
		if codeErr.Code == http.StatusNotFound || codeErr.Code == http.StatusGone {
			return validation.License{}, validation.ErrUnknownLicense
		}
		fallthrough
	default:
		return validation.License{}, fmt.Errorf("module zip request failed: %w", err)
	}

	mt, _, err := mime.ParseMediaType(rd.ContentType())
	if err != nil {
		return validation.License{}, fmt.Errorf("parse content type failed: %w", InvalidContentTypeErr(rd.ContentType()))
	}

	if mt != "application/zip" {
		return validation.License{}, InvalidContentTypeErr(mt)
	}

	moduleZIP, err := zip.NewReader(bufra.NewBufReaderAt(rd, 1024*1024), rd.Size())
	if err != nil {
		return validation.License{}, fmt.Errorf("module zip open failed: %w", err)
	}

	licMatches, err := licensedb.Detect(&ZipFiler{Reader: moduleZIP, Module: m})
	if err != nil {
		return validation.License{}, fmt.Errorf("licensedb detect failure: %w", err)
	}

	return c.licenseToReturn(m, licMatches)
}

func (c *Client) makeStore() httpreaderat.Store {
	storeMemLimit := c.StoreMemLimit
	if storeMemLimit == 0 {
		storeMemLimit = 1024 * 1024
	}

	storeFileLimit := c.StoreFileLimit
	if storeFileLimit == 0 {
		storeFileLimit = 1024 * 1024 * 1024
	}

	return httpreaderat.NewLimitedStore(
		httpreaderat.NewStoreMemory(), storeMemLimit, httpreaderat.NewLimitedStore(
			httpreaderat.NewStoreFile(), storeFileLimit, nil))
}

func (c *Client) licenseToReturn(m validation.Module, matches map[string]api.Match) (validation.License, error) {
	var (
		mostConfidentLicence string
		maxConfidence        float64
	)

	c.log.Debug(
		"license detector success",
		zap.Reflect("license_matches", matches),
		zap.Stringer("module", &m),
	)

	for name, match := range matches {
		confidence := float64(match.Confidence)
		if confidence >= c.ConfidenceThreshold && confidence > maxConfidence {
			maxConfidence = confidence
			mostConfidentLicence = name
		}
	}

	if mostConfidentLicence == "" {
		return validation.License{}, validation.ErrUnknownLicense
	}

	licInfo, _ := spdx.LicenseByID(mostConfidentLicence)

	return validation.License{
		Name:   licInfo.Name,
		SPDXID: mostConfidentLicence,
	}, nil
}
