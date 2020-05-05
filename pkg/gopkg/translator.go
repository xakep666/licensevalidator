package gopkg

import (
	"context"
	"fmt"
	"regexp"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

// re is the regexp matching the package for a GoPkg import. This is taken
// almost directly from the GoPkg source code itself so it should match
// perfectly.
var re = regexp.MustCompile(`(?i)^gopkg\.in/(?:([a-zA-Z0-9][-a-zA-Z0-9]+)/)?([a-zA-Z][-.a-zA-Z0-9]*)\.((?:v0|v[1-9][0-9]*)(?:\.0|\.[1-9][0-9]*){0,2}(?:-unstable)?)(?:\.git)?((?:/[a-zA-Z0-9][-.a-zA-Z0-9]*)*)$`)

type Translator struct{}

func (t Translator) Translate(ctx context.Context, m validation.Module) (translated validation.Module, err error) {
	ms := re.FindStringSubmatch(m.Name)
	if ms == nil {
		return m, nil
	}

	// URL case 1 with no user means it is go-<pkg>
	if ms[1] == "" {
		ms[1] = "go-" + ms[2]
	}

	// Matches, convert to github
	m.Name = fmt.Sprintf("github.com/%s/%s", ms[1], ms[2])
	return m, nil
}
