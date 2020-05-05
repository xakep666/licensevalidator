package github

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/xakep666/licensevalidator/pkg/spdx"
	"github.com/xakep666/licensevalidator/pkg/validation"

	"github.com/google/go-github/v18/github"
	"go.uber.org/zap"
	"gopkg.in/src-d/go-license-detector.v3/licensedb"
)

var githubRe = regexp.MustCompile(`^github\.com/([^/]+)/([^/]+)$`)

type ClientParams struct {
	Client *github.Client

	// FallbackConfidenceThreshold is a confidence threshold for go-license-detector fallback
	// This fallback used when github returns "other" as license name
	FallbackConfidenceThreshold float64
}

type Client struct {
	ClientParams

	log *zap.Logger
}

func NewClient(logger *zap.Logger, clientParams ClientParams) *Client {
	return &Client{
		ClientParams: clientParams,
		log:          logger.With(zap.String("component", "github-client")),
	}
}

func (c *Client) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	l := c.log.With(zap.Stringer("module", &m))
	matches := githubRe.FindStringSubmatch(m.Name)
	if len(matches) == 0 {
		l.Debug("not a github module")
		return validation.License{}, validation.ErrUnknownLicense
	}

Retry:
	rl, _, err := c.Client.Repositories.License(ctx, matches[1], matches[2])
	var rateLimitErr *github.RateLimitError
	switch {
	case errors.Is(err, nil):
		// pass
	case errors.As(err, &rateLimitErr):
		dur := time.Until(rateLimitErr.Rate.Reset.Time)
		l.Info("rate limit reached, wait", zap.Duration("wait", dur))
		timer := time.NewTimer(dur)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			// Context cancelled or ended so return early
			return validation.License{}, ctx.Err()

		case <-timer.C:
			// Rate limit should be up, retry
			goto Retry
		}

	default:
		return validation.License{}, fmt.Errorf("github failed: %w", err)
	}

	// If the license type is "other" then we try to use go-license-detector
	// to determine the license, which seems to be accurate in these cases.
	if rl.GetLicense().GetKey() == "other" {
		l.Info("github didn't detected license, trying go-license-detector")
		return c.detectFallback(m, rl)
	}

	return validation.License{
		Name:   rl.GetLicense().GetName(),
		SPDXID: rl.GetLicense().GetSPDXID(),
	}, nil
}

// detectFallback uses go-license-detector as a fallback.
func (c *Client) detectFallback(m validation.Module, rl *github.RepositoryLicense) (validation.License, error) {
	ms, err := licensedb.Detect(&filerImpl{License: rl})
	if err != nil {
		return validation.License{}, fmt.Errorf("license detector failed: %w", err)
	}

	c.log.Debug(
		"license detector success",
		zap.Reflect("license_matches", ms),
		zap.Stringer("module", &m),
	)

	var (
		highestConfidence    float64
		mostConfidentLicense string
	)
	for id, match := range ms {
		confidence := float64(match.Confidence)
		if confidence >= c.FallbackConfidenceThreshold && confidence > highestConfidence {
			highestConfidence = confidence
			mostConfidentLicense = id
		}
	}

	if mostConfidentLicense == "" {
		return validation.License{}, validation.ErrUnknownLicense
	}

	licInfo, _ := spdx.LicenseByID(mostConfidentLicense)

	return validation.License{
		Name:   licInfo.Name,
		SPDXID: mostConfidentLicense,
	}, nil
}
