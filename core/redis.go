package core

import (
	"fmt"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v2"
	"os"
)

type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password" `
	DB       int    `yaml:"db"`
}

func InitRedis() *redis.Client {
	cfg := readRedisConfig()
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Host + ":" + cfg.Port,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
}

func readRedisConfig() RedisConfig {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return RedisConfig{}
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to unmarshal config file: %v\n", err)
		return RedisConfig{}
	}

	return config.Redis
}
