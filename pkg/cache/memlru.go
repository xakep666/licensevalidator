package cache

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/xakep666/licensevalidator/pkg/validation"
)

type MemLRU struct {
	backed Cacher

	cache *lru.Cache
}

func NewMemLRU(backed Cacher, size int) (*MemLRU, error) {
	c, err := lru.New(size)
	if err != nil {
		return nil, fmt.Errorf("LRU init failed: %w", err)
	}

	return &MemLRU{
		backed: backed,
		cache:  c,
	}, nil
}

func (*MemLRU) licenseLey(m validation.Module) string {
	return fmt.Sprintf("license:%s@%s", m.Name, m.Version.Original())
}

func (ml *MemLRU) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	key := ml.licenseLey(m)
	licI, ok := ml.cache.Get(key)
	if ok {
		return licI.(validation.License), nil
	}

	lic, err := ml.backed.ResolveLicense(ctx, m)
	if err != nil {
		return validation.License{}, fmt.Errorf("%w", err)
	}

	ml.cache.Add(key, lic)
	return lic, nil
}
