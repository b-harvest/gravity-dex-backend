package config

import (
	"os"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

var DefaultConfig = Config{
	BindAddr: "0.0.0.0:8080",
	Log:      zap.NewProductionConfig(),
}

type Config struct {
	BindAddr string     `yaml:"bind_addr"`
	Log      zap.Config `yaml:"log"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	cfg := DefaultConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
