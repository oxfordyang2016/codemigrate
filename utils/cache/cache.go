package cache

import (
	clog "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"time"
)

var (
	Pool *redis.Pool
)

func newPool(server, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err

			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err

				}
			}
			return c, err

		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil

			}
			_, err := c.Do("PING")
			return err

		},
	}
}

func Init(server, password string) {
	Pool = newPool(server, password)
}

func Get() redis.Conn {
	if Pool == nil {
		clog.Critical("Please set cache pool first!")
		return nil
	}
	return Pool.Get()
}
