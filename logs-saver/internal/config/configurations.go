package config

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/tickets-dao/logs-saver/pkg/logger"
	"gopkg.in/yaml.v3"
)

const (
	localConfigFilePath = "deploy/configs/values_local.yaml"
)

// Config represents application configuration.
type Config struct {
	Listen ServerConfig
	Logger logger.Configuration `yaml:"logger"`
}

// ServerConfig represents configuration of server location
type ServerConfig struct {
	BindIP string `yaml:"bind_ip"`
	Port   string `yaml:"port"`
}

// ParseConfiguration parses configuration from values_*.yaml
func (c *Config) ParseConfiguration() error {
	c.Default()

	configFile, err := os.Open(localConfigFilePath)
	if err != nil {
		logger.Errorf(context.Background(), "failed to open config file at %s: %v", localConfigFilePath, err)
		return fmt.Errorf("failed to open config file %s: %v", localConfigFilePath, err)
	}

	data, _ := io.ReadAll(configFile)

	logger.Infof(context.Background(), "starting with config from %s", localConfigFilePath)

	return yaml.Unmarshal(data, c)
}

// Default sets default values in config variables.
func (c *Config) Default() {
	c.Listen = ServerConfig{BindIP: "0.0.0.0", Port: "12345"}
	c.Logger.Default()
}
