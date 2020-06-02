package redis

import (
	log "github.com/cihub/seelog"
	"github.com/garyburd/redigo/redis"
	"github.com/HailoOSS/service/config"
	d "github.com/HailoOSS/service/dedupe"
	"time"
)

type RedisDedupeClient struct {
	client *redis.Pool
}

func NewRedisDedupeClient() *RedisDedupeClient {
	rs := &RedisDedupeClient{}
	c, err := rs.connect()
	if err != nil {
		log.Errorf("Cannot connect to redis: %s", err)
	}
	rs.client = c
	rs.changeConfigSubscriber()
	return rs
}

func (rs *RedisDedupeClient) connect() (*redis.Pool, error) {
	host := config.AtPath("hailo", "service", "deduper", "redis", "hostname").AsString(":16379")
	// var password string
	log.Debugf("Setting redis server from config: %v", host)
	pool := &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", host)
			if err != nil {
				return nil, err
			}
			if _, err := c.Do("PING"); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return pool, nil
}

func (rs *RedisDedupeClient) changeConfigSubscriber() {
	ch := config.SubscribeChanges()

	// Listen for config changes
	go func() {
		for {
			<-ch
			rs.connect()
		}
	}()

}

func (rs *RedisDedupeClient) Add(key string, value string) error {
	conn := rs.client.Get()
	defer conn.Close()
	res, err := conn.Do("SET", key, value, "EX", 604800, "NX")
	if err != nil {
		return err
	}
	if res == nil {
		return d.ErrorKeyExistsOnWrite
	}
	return nil
}

func (rs *RedisDedupeClient) Remove(key string) error {
	conn := rs.client.Get()
	defer conn.Close()
	res, err := conn.Do("DEL", key)
	if err != nil {
		return err
	}
	if res == nil {
		return d.ErrorKeyExistsOnWrite
	}
	return nil
}

func (rs *RedisDedupeClient) Exists(key string) (bool, error) {
	conn := rs.client.Get()
	defer conn.Close()
	value, err := conn.Do("EXISTS", key)
	if err != nil {
		return false, err
	}

	if value.(int64) == 1 {
		return true, nil
	}
	if value.(int64) == 0 {
		return false, nil
	}

	return false, d.ErrorUnknown
}

func (rs *RedisDedupeClient) HealthPing() error {
	conn := rs.client.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	if err != nil {
		return err
	}

	return nil
}

func (rs *RedisDedupeClient) HealthRead(key string) error {
	conn := rs.client.Get()
	defer conn.Close()

	_, err := conn.Do("EXISTS", key)
	if err != nil {
		return err
	}

	return nil
}

func (rs *RedisDedupeClient) HealthWrite(key string) error {
	conn := rs.client.Get()
	defer conn.Close()

	value := "test"
	_, err := conn.Do("SET", key, value, "EX", 15, "NX")
	if err != nil {
		return err
	}

	return nil
}
