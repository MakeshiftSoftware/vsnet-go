package main

import (
	"time"

	"github.com/spf13/viper"
)

var defaults = map[string]interface{}{
	"EXTERNAL_IP":                 "127.0.0.1",
	"PORT":                        "8080",
	"MAX_CONNECTIONS":             255,
	"SECRET":                      "secret",
	"REDIS_QUEUE_SERVICE":         ":6379",
	"REDIS_NODE_REGISTRY_SERVICE": ":6380",
}

// Config implementation
type Config struct {
	externalIP               string
	port                     string
	maxConnections           int
	jwtSecret                []byte
	redisQueueService        string
	redisNodeRegistryService string
	launchUnixTime           int64
}

func loadConfig() *Config {
	v := viper.New()

	for key, value := range defaults {
		v.SetDefault(key, value)
	}

	v.AutomaticEnv()

	return &Config{
		externalIP:               v.GetString("EXTERNAL_IP"),
		port:                     v.GetString("PORT"),
		maxConnections:           v.GetInt("MAX_CONNECTIONS"),
		jwtSecret:                []byte(v.GetString("SECRET")),
		redisQueueService:        v.GetString("REDIS_QUEUE_SERVICE"),
		redisNodeRegistryService: v.GetString("REDIS_NODE_REGISTRY_SERVICE"),
		launchUnixTime:           time.Now().Unix(),
	}
}
