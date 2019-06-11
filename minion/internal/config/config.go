package config

import "github.com/spf13/viper"

// Environment variable names
const (
	envExternalIP     = "EXTERNAL_IP"
	envPort           = "PORT"
	envSecret         = "SECRET"
	envMaxConnections = "MAX_CONNECTIONS"
	envMaxMessageSize = "MAX_MESSAGE_SIZE"
	envRedisAddr      = "REDIS_ADDR"
)

// Default config
var defaults = map[string]interface{}{
	(envExternalIP):     ":",
	(envPort):           "8080",
	(envSecret):         "secret",
	(envMaxConnections): 255,
	(envMaxMessageSize): 512,
	(envRedisAddr):      ":6379",
}

// Config implementation
type Config struct {
	ExternalIP     string
	Port           string
	Secret         []byte
	MaxConnections int64
	MaxMessageSize int64
	RedisAddr      string
}

// New creates a new config
func New() *Config {
	v := viper.New()

	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	v.AutomaticEnv()

	return &Config{
		ExternalIP:     v.GetString(envExternalIP),
		Port:           v.GetString(envPort),
		Secret:         []byte(v.GetString(envSecret)),
		MaxConnections: v.GetInt64(envMaxConnections),
		MaxMessageSize: v.GetInt64(envMaxMessageSize),
		RedisAddr:      v.GetString(envRedisAddr),
	}
}
