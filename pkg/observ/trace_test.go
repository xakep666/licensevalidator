package observ_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/core"
	apitrace "go.opentelemetry.io/otel/api/trace"
	"go.opentelemetry.io/otel/exporters/trace/stdout"
	"go.opentelemetry.io/otel/plugin/httptrace"
	"go.opentelemetry.io/otel/plugin/othttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/xakep666/licensevalidator/pkg/observ"
)

func TestTraceTransport_RoundTrip(t *testing.T) {
	var spanBuffer bytes.Buffer

	exp, err := stdout.NewExporter(stdout.Options{
		Writer:      &spanBuffer,
		PrettyPrint: false,
	})
	require.NoError(t, err)

	tp, err := sdktrace.NewProvider(sdktrace.WithSyncer(exp))
	require.NoError(t, err)

	tracer := tp.Tracer("")

	server := httptest.NewServer(othttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEqual(t,
			apitrace.NoopSpan{},
			apitrace.SpanFromContext(r.Context()),
			"span not forwarded to server",
		)
	}),
		"test-server",
		othttp.WithTracer(tracer),
	))

	client := *server.Client()
	client.Transport = &observ.TraceTransport{
		RoundTripper: client.Transport,
		ServiceName:  "test service",
		Tracer:       tracer,
	}

	ctx, parent := tracer.Start(context.Background(), "test-parent-span")

	req, err := http.NewRequest(http.MethodGet, server.URL+"/test", nil)
	require.NoError(t, err)

	_, req = httptrace.W3C(ctx, req)

	resp, err := client.Do(req)
	parent.End()

	if assert.NoError(t, err) {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	type span struct {
		SpanKind   apitrace.SpanKind
		Attributes []map[string]interface{}
	}
	var spans []span

	t.Logf("Recorded spans:\n%s", spanBuffer.String())

	for _, rawSpan := range bytes.Split(spanBuffer.Bytes(), []byte("\n")) {
		if len(rawSpan) == 0 {
			continue
		}

		var sp span
		assert.NoError(t, json.Unmarshal(rawSpan, &sp))

		spans = append(spans, sp)
	}

	if assert.Len(t, spans, 3) { // parent-client, client-request, server-request
		assert.Equal(t, apitrace.SpanKindClient, spans[1].SpanKind)
		assert.Contains(t, spans[1].Attributes, map[string]interface{}{
			"Key": string(httptrace.URLKey),
			"Value": map[string]interface{}{
				"Type":  core.STRING.String(),
				"Value": server.URL + "/test",
			},
		})
		assert.Contains(t, spans[1].Attributes, map[string]interface{}{
			"Key": string(httptrace.HTTPStatus),
			"Value": map[string]interface{}{
				"Type":  core.INT64.String(),
				"Value": float64(http.StatusOK),
			},
		})
		assert.Contains(t, spans[1].Attributes, map[string]interface{}{
			"Key": "http.method",
			"Value": map[string]interface{}{
				"Type":  core.STRING.String(),
				"Value": http.MethodGet,
			},
		})
	}
}
