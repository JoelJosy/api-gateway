package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port       int         `yaml:"port"`
	Redis      RedisConfig `yaml:"redis"`
	RateLimit  RateLimit   `yaml:"rate_limit"`
	Proxy      ProxyConfig `yaml:"proxy"`
	Routes     []Route     `yaml:"routes"`
	PubKeyPath string      `yaml:"public_key_path"`
}

type Route struct {
	Path         string `yaml:"path"`
	Upstream     string `yaml:"upstream"`
	AuthRequired bool   `yaml:"auth_required"`
}
type RedisConfig struct {
	Address string `yaml:"address"`
}

type ProxyConfig struct {
	DialTimeout             time.Duration `yaml:"dial_timeout"`
	TLSHandshakeTimeout     time.Duration `yaml:"tls_handshake_timeout"`
	ResponseHeaderTimeout   time.Duration `yaml:"response_header_timeout"`
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold"`
	CircuitBreakerCooldown  time.Duration `yaml:"circuit_breaker_cooldown"`
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
