package config

import (
	"fmt"

	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

type ServerConfig struct {
	Metrics   bool
	Profiling bool
}

var serverConfig = ServerConfig{}

func GetServerConfig() ServerConfig {
	return serverConfig
}

func InitServerConfig() (ServerConfig, error) {
	config.WithOptions(config.ParseEnv, config.ParseDefault)
	config.AddDriver(yaml.Driver)

	temp := map[string]any{
		"metrics":     "false",
		"profiling":   "false",
		"keybindings": map[string]string{},
	}
	err := config.LoadData(temp)
	if err != nil {
		return serverConfig, fmt.Errorf("error loading data: %w", err)
	}

	err = config.LoadExists("server.yaml", "~/.khronoscope/server.yaml", "../../server.yaml")
	if err != nil {
		return serverConfig, fmt.Errorf("error loading config file: %w", err)
	}
	err = config.Decode(&serverConfig)
	if err != nil {
		return serverConfig, fmt.Errorf("error decoding config data: %w", err)
	}

	return serverConfig, nil
}
