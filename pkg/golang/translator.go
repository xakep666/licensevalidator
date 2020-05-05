package golang

import (
	"context"
	"fmt"
	"regexp"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

var re = regexp.MustCompile(`^(go\.googlesource\.com|golang\.org/x)/([^/]+)$`)

type Translator struct{}

func (t Translator) Translate(ctx context.Context, m validation.Module) (translated validation.Module, err error) {
	ms := re.FindStringSubmatch(m.Name)
	if ms == nil {
		return m, nil
	}

	m.Name = fmt.Sprintf("github.com/golang/%s", ms[2])
	return m, nil
}
