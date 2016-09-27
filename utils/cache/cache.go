package cache

import (
	clog "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"time"
)

var (
	Pool *redis.Pool
)

func newPool(url string, max_idle, idle_timeout int) *redis.Pool {
	timeout := time.Duration(idle_timeout) * time.Second
	return &redis.Pool{
		MaxIdle:     max_idle,
		IdleTimeout: timeout,
		Dial: func() (redis.Conn, error) {
			c, err := redis.DialURL(url)
			if err != nil {
				return nil, err

			}
			// if password != "" {
			// 	if _, err := c.Do("AUTH", password); err != nil {
			// 		c.Close()
			// 		return nil, err
			//
			// 	}
			// }
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

func Init(url string, max_idle, idle_timeout int) {
	Pool = newPool(url, max_idle, idle_timeout)
}

func Get() redis.Conn {
	if Pool == nil {
		clog.Critical("Please set cache pool first!")
		return nil
	}
	return Pool.Get()
}

func Ping() error {
	conn := Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}
