package core

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type SystemConfig struct {
	SelfAddress    string `yaml:"self_address"`
	MonitorAddress string `yaml:"monitor_address"`
}

func InitSystemConfig() SystemConfig {
	return readSystemConfig()
}

func readSystemConfig() SystemConfig {
	data, err := os.ReadFile("config.yml")
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		return SystemConfig{}
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("Failed to unmarshal config file: %v\n", err)
		return SystemConfig{}
	}

	return config.SystemConfig
}
