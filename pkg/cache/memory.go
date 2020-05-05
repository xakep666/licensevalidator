package cache

import (
	"context"
	"fmt"
	"sync"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

// MemoryCache is a simple in-memory cache that holds license for project version
// It's needed because sometimes license recognition takes a lot of time
type MemoryCache struct {
	Backed Cacher
	m      sync.Map
}

func (*MemoryCache) licenseLey(m validation.Module) string {
	return fmt.Sprintf("license:%s@%s", m.Name, m.Version.Original())
}

func (c *MemoryCache) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	key := c.licenseLey(m)
	item, ok := c.m.Load(key)
	if ok {
		return item.(validation.License), nil
	}

	lic, err := c.Backed.ResolveLicense(ctx, m)
	if err != nil {
		return validation.License{}, fmt.Errorf("%w", err)
	}

	c.m.Store(key, lic)
	return lic, nil
}
