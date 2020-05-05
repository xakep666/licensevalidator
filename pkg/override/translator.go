// Package override contains a translator and license resolver using a raw map[string]string
package override

import (
	"context"
	"regexp"

	"github.com/xakep666/licensevalidator/pkg/validation"

	"go.uber.org/zap"
)

type TranslateOverride struct {
	Match   *regexp.Regexp
	Replace string
}

type Translator struct {
	overrides []TranslateOverride
	log       *zap.Logger
}

func NewTranslator(log *zap.Logger, overrides []TranslateOverride) *Translator {
	return &Translator{overrides: overrides, log: log.With(zap.String("component", "override_translator"))}
}

func (t Translator) Translate(ctx context.Context, m validation.Module) (validation.Module, error) {
	for _, override := range t.overrides {
		if override.Match.MatchString(m.Name) {
			repl := override.Match.ReplaceAllString(m.Name, override.Replace)
			t.log.Debug("override replacement", zap.String("original", m.Name), zap.String("replaced", repl))
			m.Name = repl
			return m, nil
		}
	}

	return m, nil
}
