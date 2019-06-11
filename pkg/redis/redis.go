package redis

import (
	"time"

	"github.com/cenkalti/backoff"
	"github.com/garyburd/redigo/redis"
)

// Client implementation
type Client struct {
	Pool *redis.Pool
}

// New creates a new redis client with connection pool
func New(addr string) *Client {
	return &Client{
		Pool: &redis.Pool{
			MaxIdle:     10,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr)
			},
			TestOnBorrow: func(conn redis.Conn, t time.Time) error {
				_, err := conn.Do("PING")
				return err
			},
		},
	}
}

// WaitForConnection pings redis with an exponential backoff
// to wait until connection is made
func (c *Client) WaitForConnection() error {
	return backoff.Retry(func() error {
		return c.Ping()
	}, backoff.NewExponentialBackOff())
}

// Close closes the connection pool
func (c *Client) Close() error {
	return c.Pool.Close()
}

// Ping pings redis
func (c *Client) Ping() error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("PING")
	return err
}

// Get gets value of key
func (c *Client) Get(key interface{}) (interface{}, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return conn.Do("GET", key)
}

// Set sets key to value
func (c *Client) Set(args ...interface{}) (bool, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	res, err := conn.Do("SET", args...)
	return res == "OK", err
}

// Delete deletes key
func (c *Client) Delete(key interface{}) error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("DEL", key)
	return err
}

// Exists checks if key exists
func (c *Client) Exists(key interface{}) (bool, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("EXISTS", key))
}

// Incr increments key
func (c *Client) Incr(key interface{}) (int, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("INCR", key))
}

// Incrby increments key by amount
func (c *Client) Incrby(key interface{}, amount int) (int, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("INCRBY", key, amount))
}

// Expire expires key in seconds
func (c *Client) Expire(key interface{}, time int) (bool, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("EXPIRE", key, time))
}

// Pexpire expires key in milliseconds
func (c *Client) Pexpire(key interface{}, time int) (bool, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("PEXPIRE", key, time))
}

// Hget get field value of hash at key
func (c *Client) Hget(key interface{}, field interface{}) (interface{}, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return conn.Do("HGET", key, field)
}

// Hset sets field in the hash stored at key to value
func (c *Client) Hset(key interface{}, field interface{}, value interface{}) error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", key, field, value)
	return err
}

// Hincrby increments field of hash stored at key by amount
func (c *Client) Hincrby(key interface{}, field interface{}, amount int32) (interface{}, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return conn.Do("HINCRBY", key, field, amount)
}

// Hgetall get all fields and values of the hash stored at key
func (c *Client) Hgetall(key interface{}) ([]interface{}, error) {
	conn := c.Pool.Get()
	defer conn.Close()
	return redis.Values(conn.Do("HGETALL", key))
}

// Hmset sets the specified fields to their respective values in the hash stored at key
func (c *Client) Hmset(args ...interface{}) error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("HMSET", args...)
	return err
}

// Rpush right pushes value into list
func (c *Client) Rpush(key interface{}, value interface{}) error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("RPUSH", key, value)
	return err
}

// Lpush left pushes value into list
func (c *Client) Lpush(key interface{}, value interface{}) error {
	conn := c.Pool.Get()
	defer conn.Close()
	_, err := conn.Do("LPUSH", key, value)
	return err
}

// GetKeys get all keys that match pattern
func (c *Client) GetKeys(pattern string) ([]string, error) {
	conn := c.Pool.Get()
	defer conn.Close()

	iter := 0
	keys := []string{}
	included := map[string]bool{}

	for {
		arr, err := redis.Values(conn.Do("SCAN", iter, "MATCH", pattern))

		if err != nil {
			return keys, err
		}

		iter, _ = redis.Int(arr[0], nil)
		k, _ := redis.Strings(arr[1], nil)

		for _, s := range k {
			if !included[s] {
				included[s] = true
				keys = append(keys, s)
			}
		}

		if iter == 0 {
			break
		}
	}

	return keys, nil
}
