package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port   int     `yaml:"port"`
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
	Auth_Required bool `yaml: "auth_required"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}