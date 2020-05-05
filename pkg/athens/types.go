package athens

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
)

type ValidationRequest struct {
	Module  string
	Version *semver.Version
}

type Validator interface {
	Validate(ctx context.Context, req ValidationRequest) error
}

// ErrForbidden should be returned by Validator if module validation failed by rule set
type ErrForbidden struct {
	Inner error
}

func (e *ErrForbidden) Error() string { return fmt.Sprintf("module forbidden: %s", e.Inner) }

func (e *ErrForbidden) Unwrap() error { return e.Inner }
