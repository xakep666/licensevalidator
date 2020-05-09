package cache

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/mediocregopher/radix/v3"
	"github.com/mediocregopher/radix/v3/resp/resp2"

	"github.com/xakep666/licensevalidator/pkg/validation"
)

type RedisCache struct {
	Backed Cacher
	Client radix.Client
	TTL    time.Duration
}

func (*RedisCache) licenseKey(m validation.Module) string {
	return fmt.Sprintf("licensevalidator:license:%s@%s", m.Name, m.Version.Original())
}

func (rc *RedisCache) ResolveLicense(ctx context.Context, m validation.Module) (validation.License, error) {
	key := rc.licenseKey(m)

	var ret validation.License
	maybeEmpty := MaybeEmpty{Rcv: &ret}

	err := rc.Client.Do(radix.Cmd(&maybeEmpty, "HGETALL", key))
	if err != nil {
		return ret, fmt.Errorf("get license from redis failed: %w", err)
	}

	if !maybeEmpty.Empty {
		return ret, nil
	}

	ret, err = rc.Backed.ResolveLicense(ctx, m)
	if err != nil {
		return ret, fmt.Errorf("%w", err)
	}

	cmds := []radix.CmdAction{
		radix.FlatCmd(nil, "HMSET", key, ret),
	}
	if rc.TTL > 0 {
		cmds = append(cmds, radix.FlatCmd(nil, "PEXPIRE", key, int64(rc.TTL/time.Millisecond)))
	}

	err = rc.Client.Do(radix.Pipeline(cmds...))
	if err != nil {
		return ret, fmt.Errorf("set license in redis failed: %w", err)
	}

	return ret, nil
}

// MaybeEmpty is a helper for fetching not always existing keys
// radix.MaybeNil is not suitable for HGETALL currently
type MaybeEmpty struct {
	Empty bool
	Rcv   interface{}
}

func (mn *MaybeEmpty) UnmarshalRESP(br *bufio.Reader) error {
	rm := make(resp2.RawMessage, 0)
	if err := rm.UnmarshalRESP(br); err != nil {
		return err
	} else if bytes.Equal(rm, []byte("*0\r\n")) {
		mn.Empty = true
		return nil
	}

	return rm.UnmarshalInto(resp2.Any{I: mn.Rcv})
}
