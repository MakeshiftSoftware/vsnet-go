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

// NewClient creates a new redis client with connection pool
func NewClient(addr string) *Client {
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
func (c *Client) GetString(key string) (string, error) {
	con := c.Pool.Get()
	defer con.Close()
	data, err := redis.String(con.Do("GET", key))
	return data, err
}

// GetInt gets value of key as an int64
func (c *Client) GetInt(key string) (int64, error) {
	con := c.Pool.Get()
	defer con.Close()
	data, err := redis.Int64(con.Do("GET", key))
	return data, err
}

// GetBool gets value of key as a bool
func (c *Client) GetBool(key string) (bool, error) {
	con := c.Pool.Get()
	defer con.Close()
	data, err := redis.Bool(con.Do("GET", key))
	return data, err
}

// Set sets key to value
func (c *Client) Set(key string, value string) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("SET", key, value)
	return err
}

// Exists checks if key exists
func (c *Client) Exists(key string) (bool, error) {
	con := c.Pool.Get()
	defer con.Close()
	ok, err := redis.Bool(con.Do("EXISTS", key))
	return ok, err
}

// Delete deletes key
func (c *Client) Delete(key string) error {
	con := c.Pool.Get()
	defer con.Close()
	_, err := con.Do("DEL", key)
	return err
}

// Incr increments key
func (c *Client) Incr(key string) (int, error) {
	con := c.Pool.Get()
	defer con.Close()
	data, err := redis.Int(con.Do("INCR", key))
	return data, err
}

// Hset sets field in the hash stored at key to value
func (c *Client) Hset(key string, field string, value string) error {
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

// Hgetall get all fields and values of the hash stored at key
func (c *Client) Hgetall(key string) ([]interface{}, error) {
	con := c.Pool.Get()
	defer con.Close()
	data, err := redis.Values(con.Do("HGETALL", key))
	return data, err
}
