package config

import "github.com/spf13/viper"

const (
	envExternalIP         = "EXTERNAL_IP"
	envPort               = "PORT"
	envSecret             = "SECRET"
	envMaxConnections     = "MAX_CONNECTIONS"
	envMaxMessageSize     = "MAX_MESSAGE_SIZE"
	envRedisBrokerAddr    = "REDIS_BROKER_ADDR"
	envRedisRegistrarAddr = "REDIS_REGISTRAR_ADDR"
	envRedisServerPrefix  = "REDIS_SERVER_PREFIX"
)

var defaults = map[string]interface{}{
	(envExternalIP):         ":",
	(envPort):               "8080",
	(envSecret):             "secret",
	(envMaxConnections):     255,
	(envMaxMessageSize):     512,
	(envRedisBrokerAddr):    ":6379",
	(envRedisRegistrarAddr): ":6380",
	(envRedisServerPrefix):  "Server:",
}

// Config implementation
type Config struct {
	ExternalIP         string
	Port               string
	Secret             []byte
	MaxConnections     int64
	MaxMessageSize     int64
	RedisBrokerAddr    string
	RedisRegistrarAddr string
	RedisServerPrefix  string
}

// New creates a new config
func New() *Config {
	v := viper.New()

	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	v.AutomaticEnv()

	return &Config{
		ExternalIP:         v.GetString(envExternalIP),
		Port:               v.GetString(envPort),
		Secret:             []byte(v.GetString(envSecret)),
		MaxConnections:     v.GetInt64(envMaxConnections),
		MaxMessageSize:     v.GetInt64(envMaxMessageSize),
		RedisBrokerAddr:    v.GetString(envRedisBrokerAddr),
		RedisRegistrarAddr: v.GetString(envRedisRegistrarAddr),
		RedisServerPrefix:  v.GetString(envRedisServerPrefix),
	}
}
