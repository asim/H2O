/**
 * This is a healthcheck for the Amazon AWS ElastiCache Service
 */

package aws_healthchecks_ec

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/HailoOSS/service/healthcheck"
	"time"
)

const HealthCheckId = "com.HailoOSS.service.aws_ec"

type ECClient struct {
	conn redis.Conn
}

// HealthCheck asserts we can connect to elasticache and we can read and write items
func HealthCheck(host string) healthcheck.Checker {
	return func() (map[string]string, error) {
		key := fmt.Sprintf("hailo-healthcheck-%d", time.Now().UTC().UnixNano())
		ec, err := newECClient(host)

		if err != nil {
			return nil, err
		}

		if err := ec.HealthPing(); err != nil {
			return nil, err
		}

		if err := ec.HealthWrite(key); err != nil {
			return nil, err
		}

		if err := ec.HealthRead(key); err != nil {
			return nil, err
		}

		ec.Disconnect()

		return nil, nil
	}
}

func newECClient(host string) (*ECClient, error) {
	ec := &ECClient{}
	c, err := redis.Dial("tcp", host)
	if err != nil {
		return ec, err
	}

	if _, err := c.Do("PING"); err != nil {
		c.Close()
		return ec, err
	}

	ec.conn = c

	return ec, nil
}

func (ec *ECClient) HealthPing() error {
	_, err := ec.conn.Do("PING")
	if err != nil {
		return err
	}

	return nil
}

func (ec *ECClient) HealthRead(key string) error {
	_, err := ec.conn.Do("EXISTS", key)
	if err != nil {
		return err
	}

	return nil
}

func (ec *ECClient) HealthWrite(key string) error {
	value := "test"
	_, err := ec.conn.Do("SET", key, value, "EX", 15, "NX")
	if err != nil {
		return err
	}

	return nil
}

func (ec *ECClient) Disconnect() {
	ec.conn.Close()
}
