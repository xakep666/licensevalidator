package testutil

import (
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

// TLogConsumer is a testing.T-based log consumer for testcontainers-go with prefix in logs
type TLogConsumer struct {
	*testing.T
	Prefix string
}

func (c *TLogConsumer) Accept(log testcontainers.Log) {
	c.Logf("%s [%s]: %s", c.Prefix, log.LogType, log.Content)
}
