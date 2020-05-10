package observ

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

type LicenseResolver struct {
	validation.LicenseResolver
	Meter metric.Meter

	initMetricsOnce sync.Once
	licenseMetric   metric.Int64Counter
}

func (l *LicenseResolver) initMetrics() {
	l.initMetricsOnce.Do(func() {
		m := l.Meter
		if m == nil {
			m = metric.NoopMeter{}
		}

		l.licenseMetric, _ = m.NewInt64Counter("detected_licenses", metric.WithDescription("Count of detected licenses by name/ID"))
	})
}

func (l *LicenseResolver) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	l.initMetrics()

	lic, err := l.LicenseResolver.ResolveLicense(ctx, m)
	switch {
	case errors.Is(err, nil):
		l.licenseMetric.Add(ctx, 1, key.String("name", lic.Name), key.String("id", lic.SPDXID))
	case errors.Is(err, validation.ErrUnknownLicense):
		l.licenseMetric.Add(ctx, 1, key.String("name", "unknown"), key.String("id", "unknown"))
	}

	return lic, err
}
