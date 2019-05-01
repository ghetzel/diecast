package diecast

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gomodule/redigo/redis"
)

var redisConnectionPool sync.Map
var redisPoolMaxIdle = 10
var redisPoolIdleTimeout = 120 * time.Second
var redisPoolMaxLifetime = 10 * time.Minute

type RedisProtocol struct {
}

func (self *RedisProtocol) Retrieve(rr *ProtocolRequest) (*ProtocolResponse, error) {
	pid := rr.URL.Host

	if pid == `` {
		pid = rr.Conf(`redis`, `default_host`, `localhost:6379`).String()
	}

	if rr.Verb == `` {
		rr.Verb = `GET`
	} else {
		rr.Verb = strings.ToUpper(rr.Verb)
	}

	var pool *redis.Pool

	if v, ok := redisConnectionPool.Load(pid); ok {
		if p, ok := v.(*redis.Pool); ok {
			pool = p
		}
	}

	if pool == nil {
		pool = &redis.Pool{
			MaxIdle:         int(rr.Conf(`redis`, `max_idle`, redisPoolMaxIdle).Int()),
			IdleTimeout:     rr.Conf(`redis`, `idle_timeout`, redisPoolIdleTimeout).Duration(),
			MaxConnLifetime: rr.Conf(`redis`, `max_lifetime`, redisPoolMaxLifetime).Duration(),
			Dial: func() (redis.Conn, error) {
				return redis.Dial(`tcp`, pid)
			},
		}

		log.Debugf("RedisProtocol: created new pool to handle connections to %s", pid)
		redisConnectionPool.Store(pid, pool)
	}

	// setup context and load it up with cancel functions and timeouts and cool stuff like that
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	if timeout := typeutil.V(rr.Binding.Timeout).Duration(); timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}

	defer cancel()

	if conn, err := pool.GetContext(ctx); err == nil {
		defer conn.Close()

		// conn.Do(commandName string, args ...interface{}) (reply interface{}, err error)
		return nil, fmt.Errorf("Not Implemented")
	} else {
		return nil, fmt.Errorf("Cannot obtain Redis connection: %v", err)
	}
}
