package athens

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type ValidatorMock struct {
	mock.Mock
}

func (m *ValidatorMock) Validate(ctx context.Context, req ValidationRequest) error {
	return m.Called(ctx, req).Error(0)
}
