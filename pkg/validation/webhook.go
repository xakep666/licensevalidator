package validation

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type WebhookNotifierParams struct {
	Client       *http.Client
	Address      string
	Method       string // default is POST
	Headers      map[string]string
	BodyTemplate *template.Template
}

// WebhookNotifier calls http endpoint when license for projects with unknown license
type WebhookNotifier struct {
	WebhookNotifierParams

	log *zap.Logger
}

func NewWebhookNotifier(log *zap.Logger, webhookNotifierParams WebhookNotifierParams) *WebhookNotifier {
	return &WebhookNotifier{
		WebhookNotifierParams: webhookNotifierParams,
		log:                   log.With(zap.String("component", "webhook_notifier")),
	}
}

// WebhookTemplateContext is a request body template execution context
type WebhookTemplateContext struct {
	Module Module
}

func (w *WebhookNotifier) NotifyUnknownLicense(ctx context.Context, m Module) error {
	client := w.Client
	if client == nil {
		client = http.DefaultClient
	}

	pr, pw := io.Pipe()
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		defer pw.Close()

		return w.BodyTemplate.Execute(pw, WebhookTemplateContext{
			Module: m,
		})
	})

	eg.Go(func() error {
		defer pr.Close()

		method := http.MethodPost
		if w.Method != "" {
			method = w.Method
		}

		req, err := http.NewRequest(method, w.Address, pr)
		if err != nil {
			return fmt.Errorf("http request construct failed: %w", err)
		}

		req = req.WithContext(egCtx)

		for k, v := range w.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}

		defer resp.Body.Close()

		var bodyBuf bytes.Buffer

		_, err = io.Copy(&bodyBuf, io.LimitReader(resp.Body, 1024)) // limit size to 1k to prevent log bloat
		if err != nil {
			return fmt.Errorf("read body failed: %w", err)
		}

		w.log.Debug("Webhook response", zap.Int("code", resp.StatusCode), zap.Stringer("body", &bodyBuf))

		if resp.StatusCode >= http.StatusBadRequest {
			return fmt.Errorf("server returned bad status %d with body: %s", resp.StatusCode, bodyBuf.String())
		}

		return nil
	})

	return eg.Wait()
}
