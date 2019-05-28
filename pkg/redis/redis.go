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
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				return redis.Dial("tcp", addr)
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
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
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("PING")
	return err
}

// GetString gets value of key as a string
func (c *Client) GetString(key interface{}) (string, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.String(con.Do("GET", key))
}

// GetInt gets value of key as an int64
func (c *Client) GetInt(key interface{}) (int64, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Int64(con.Do("GET", key))
}

// GetBool gets value of key as a bool
func (c *Client) GetBool(key interface{}) (bool, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Bool(con.Do("GET", key))
}

// Set sets key to value
func (c *Client) Set(key interface{}, value interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("SET", key, value)
	return err
}

// Exists checks if key exists
func (c *Client) Exists(key interface{}) (bool, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Bool(con.Do("EXISTS", key))
}

// Delete deletes key
func (c *Client) Delete(key interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("DEL", key)
	return err
}

// Incr increments key
func (c *Client) Incr(key interface{}) (int, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Int(con.Do("INCR", key))
}

// Hset sets field in the hash stored at key to value
func (c *Client) Hset(key interface{}, field interface{}, value interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("HSET", key, field, value)
	return err
}

// Hmset sets the specified fields to their respective values in the hash stored at key
func (c *Client) Hmset(args []interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("HMSET", args...)
	return err
}

// HgetString get field of hash stored at key as string
func (c *Client) HgetString(key interface{}, field interface{}) (string, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.String(con.Do("HGET", key, field))
}

// HgetInt get field of hash stored at key as int
func (c *Client) HgetInt(key interface{}, field interface{}) (int, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Int(con.Do("HGET", key, field))
}

// Hgetall get all fields and values of the hash stored at key
func (c *Client) Hgetall(key interface{}) ([]interface{}, error) {
	con := c.Pool.Get()
	defer con.Close()
	return redis.Values(con.Do("HGETALL", key))
}

// Hincrby increments field of hash stored at key by amount
func (c *Client) Hincrby(key interface{}, field interface{}, amount uint32) (interface{}, error) {
	con := c.Pool.Get()
	defer con.Close()
	return con.Do("HINCRBY", key, field, amount)
}

// Rpush right pushes into list
func (c *Client) Rpush(key interface{}, data interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("RPUSH", key, data)
	return err
}

// Lpush left pushes into list
func (c *Client) Lpush(key interface{}, data interface{}) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("LPUSH", key, data)
	return err
}
