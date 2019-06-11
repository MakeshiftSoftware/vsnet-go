package config

import "github.com/spf13/viper"

// Environment variable names
const (
	envPort           = "PORT"
	envMaxConnections = "MAX_CONNECTIONS"
	envRedisAddr      = "REDIS_ADDR"
)

// Default config
var defaults = map[string]interface{}{
	(envPort):           "8081",
	(envMaxConnections): 255,
	(envRedisAddr):      ":6379",
}

// Config implementation
type Config struct {
	Port           string // Master port
	MaxConnections uint64 // Max connections per minion
	RedisAddr      string // Redis connection string
}

// New creates a new config
func New() *Config {
	v := viper.New()

	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	v.AutomaticEnv()

	return &Config{
		Port:           v.GetString(envPort),
		MaxConnections: v.GetUint64(envMaxConnections),
		RedisAddr:      v.GetString(envRedisAddr),
	}
}
