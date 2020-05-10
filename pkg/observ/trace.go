package observ

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

// TraceTransport is a wrapper of http.RoundTripper for tracing requests through opentelemetry
type TraceTransport struct {
	http.RoundTripper

	ServiceName string
	Tracer      trace.Tracer
	Meter       metric.Meter

	metricsInitOnce sync.Once
	durationMetric  metric.Float64Measure
}

func (t *TraceTransport) initMetrics() {
	t.metricsInitOnce.Do(func() {
		m := t.Meter
		if m == nil {
			m = metric.NoopMeter{}
		}

		t.durationMetric, _ = m.NewFloat64Measure("http_request_duration", metric.WithDescription("HTTP Request duration"))
	})
}

func (t *TraceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.initMetrics()

	transport := http.DefaultTransport
	if t.RoundTripper != nil {
		transport = t.RoundTripper
	}

	ctx, span := t.Tracer.Start(
		request.Context(),
		fmt.Sprintf("HTTP request to %s", t.ServiceName),
		trace.WithAttributes(
			httptrace.URLKey.String(request.URL.String()),
			httptrace.HostKey.String(request.Header.Get("Host")),
			httptrace.HTTPRemoteAddr.String(request.RemoteAddr),
			key.String("http.method", request.Method),
			httptrace.HTTPHeaderMIME.String(request.Header.Get("Content-Type")),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)

	httptrace.Inject(ctx, request)
	ctx, request = httptrace.W3C(ctx, request)

	defer span.End()

	start := time.Now()
	response, err := transport.RoundTrip(request)

	if err != nil {
		span.RecordError(ctx, err)
		t.durationMetric.Record(ctx, time.Since(start).Seconds(),
			httptrace.HostKey.String(request.Header.Get("Host")),
			key.String("service", t.ServiceName),
			key.String("http.method", request.Method),
			key.Bool("error", true),
		)
		return nil, err
	}

	span.SetAttributes(httptrace.HTTPStatus.Int(response.StatusCode))
	t.durationMetric.Record(ctx, time.Since(start).Seconds(),
		httptrace.HostKey.String(request.Header.Get("Host")),
		key.String("service", t.ServiceName),
		key.String("http.method", request.Method),
		key.Bool("error", false),
		httptrace.HTTPStatus.Int(response.StatusCode),
	)

	return response, nil
}
