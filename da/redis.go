package da

import (
	"github.com/garyburd/redigo/redis"
	"time"
)

func OpenRedis() redis.Conn {
	return pool.Get()
}

var pool *redis.Pool

func init() {
	pool = &redis.Pool{
		Dial:        func() (redis.Conn, error) { return redis.Dial("tcp", "localhost:6379") },
		MaxIdle:     10,
		IdleTimeout: 240 * time.Second,
		Wait:        true,
	}
}
