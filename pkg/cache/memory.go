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

	licenseMu          sync.RWMutex
	licenseMapOnceInit sync.Once
	licenseMap         map[string]validation.License
}

func (*MemoryCache) licenseLey(m validation.Module) string {
	return fmt.Sprintf("license:%s@%s", m.Name, m.Version.Original())
}

func (c *MemoryCache) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	c.licenseMapOnceInit.Do(func() {
		c.licenseMap = make(map[string]validation.License)
	})

	key := c.licenseLey(m)
	c.licenseMu.RLock()
	item, ok := c.licenseMap[key]
	c.licenseMu.RUnlock()
	if ok {
		return item, nil
	}

	lic, err := c.Backed.ResolveLicense(ctx, m)
	if err != nil {
		return validation.License{}, fmt.Errorf("%w", err)
	}

	c.licenseMu.Lock()
	c.licenseMap[key] = lic
	c.licenseMu.Unlock()

	return lic, nil
}
