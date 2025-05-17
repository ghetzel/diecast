package diecast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gomodule/redigo/redis"
)

var redisConnectionPool sync.Map
var redisPoolMaxIdle = 10
var redisPoolIdleTimeout = 120 * time.Second
var redisPoolMaxLifetime = 10 * time.Minute
var redisBlacklistedCommands = []string{
	`AUTH`,
	`BGREWRITEAOF`,
	`BGSAVE`,
	`CLIENT`,
	`CLUSTER`,
	`COMMAND`,
	`DBSIZE`,
	`DEBUG`,
	`DUMP`,
	`ECHO`,
	`EVALSHA`,
	`EVAL`,
	`FLUSHALL`,
	`FLUSHALL`,
	`FLUSHDB`,
	`INFO`,
	`KEYS`,
	`MEMORY`,
	`MIGRATE`,
	`MONITOR`,
	`OBJECT`,
	`PING`,
	`PSUBSCRIBE`,
	`PUBLISH`,
	`PUBSUB`,
	`PUNSUBSCRIBE`,
	`QUIT`,
	`RANDOMKEY`,
	`REPLICAOF`,
	`RESTORE`,
	`ROLE`,
	`SAVE`,
	`SCAN`,
	`SCRIPT`,
	`SELECT`,
	`SHUTDOWN`,
	`SLAVEOF`,
	`SLOWLOG`,
	`SUBSCRIBE`,
	`SWAPDB`,
	`SYNC`,
	`TYPE`,
	`UNSUBSCRIBE`,
	`UNWATCH`,
	`WAIT`,
	`WATCH`,
}

// The Redis binding protocol is used to retrieve or modify items in a Redis server.
// It is specified with URLs that use the redis://[host[:port]]/db/key scheme.
//
// # Protocol Options
//
//   - redis.default_host (localhost:6379)
//     Specifies the hostname:port to use if a resource URI does not specify one.
//
//   - redis.max_idle (10)
//     The maximum number of idle connections to maintain in a connection pool.
//
//   - redis.idle_timeout (120s)
//     The maximum idle time of a connection before it is closed.
//
//   - redis.max_lifetime (10m)
//     The maximum amount of time a connection can remain open before being recycled.
type RedisProtocol struct {
}

func (protocol *RedisProtocol) Retrieve(rr *ProtocolRequest) (*ProtocolResponse, error) {
	var pid = rr.URL.Host

	if pid == `` {
		pid = rr.Conf(`redis`, `default_host`, `localhost:6379`).String()
	}

	if rr.Verb == `` {
		rr.Verb = `GET`
	} else {
		rr.Verb = strings.ToUpper(rr.Verb)
	}

	if first, _ := stringutil.SplitPair(rr.Verb, ` `); sliceutil.ContainsString(redisBlacklistedCommands, first) {
		return nil, fmt.Errorf("the %q command is not permitted", rr.Verb)
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
	var ctx = context.Background()
	ctx, cancel := context.WithCancel(ctx)

	if timeout := typeutil.V(rr.Binding.Timeout).Duration(); timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}

	defer cancel()

	if conn, err := pool.GetContext(ctx); err == nil {
		defer conn.Close()

		var args = strings.Split(strings.TrimPrefix(rr.URL.Path, `/`), `/`)
		args = sliceutil.CompactString(args)

		if reply, err := conn.Do(rr.Verb, sliceutil.Sliceify(args)...); err == nil {
			var buf = bytes.NewBuffer(nil)
			var response = &ProtocolResponse{
				Raw:        reply,
				StatusCode: 200,
				data:       io.NopCloser(buf),
			}

			switch replytyped := reply.(type) {
			case error:
				return response, replytyped
			case int64, string, []byte:
				response.MimeType = `text/plain; charset=utf-8`
				buf.Write([]byte(typeutil.String(reply)))

			case []any:
				response.MimeType = `application/json; charset=utf-8`

				// handles H-series replies (maps represented as arrays of alternating keys, values)
				if strings.HasPrefix(rr.Verb, `H`) {
					var obj = make(map[string]any)
					var values = sliceutil.Stringify(replytyped)

					for i, value := range values {
						if i%2 == 0 {
							if (i + 1) < len(values) {
								obj[value] = values[i+1]
							} else {
								obj[value] = nil
							}
						}
					}

					reply = obj
				} else {
					var values = replytyped

					switch rr.Verb {
					case `TIME`:
						reply = time.Unix(typeutil.Int(values[0]), typeutil.Int(values[1]))
					default:
						reply = sliceutil.Autotype(values)
					}
				}

				if err := json.NewEncoder(buf).Encode(reply); err != nil {
					return response, err
				}
			default:
				return nil, fmt.Errorf("unsupported response type %T", reply)
			}

			return response, nil
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("cannot obtain Redis connection: %v", err)
	}
}
