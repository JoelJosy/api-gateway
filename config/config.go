package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port   int `yaml:"port"`
	Redis  RedisConfig `yaml:"redis"`
	RateLimit  RateLimit `yaml:"rate_limit"`
	Routes []Route `yaml:"routes"`
}

type Route struct {
	Path     string `yaml:"path"`
	Upstream string `yaml:"upstream"`
	AuthRequired bool `yaml:"auth_required"`
}
type RedisConfig struct {
	Address string `yaml:"address"`
}
type RateLimit struct {

	MaxTokens  int    `yaml:"max_tokens"`
	RefillRate int    `yaml:"refill_rate"`
	KeyBy      string `yaml:"key_by"` // "ip" or "user_id"

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