package observ

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/plugin/httptrace"
)

// TraceTransport is a wrapper of http.RoundTripper for tracing requests through opentelemetry
type TraceTransport struct {
	http.RoundTripper

	ServiceName string
	Tracer      trace.Tracer
}

func (t *TraceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
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

	response, err := transport.RoundTrip(request)
	if err != nil {
		span.RecordError(ctx, err)
		return nil, err
	}

	span.SetAttributes(httptrace.HTTPStatus.Int(response.StatusCode))

	return response, nil
}
