package core

type Config struct {
	Redis        RedisConfig  `yaml:"redis"`
	SystemConfig SystemConfig `yaml:"system"`
}
