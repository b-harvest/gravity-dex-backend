package config

import (
	"os"

	"gopkg.in/yaml.v2"
)

var DefaultConfig = Config{
	Server:      DefaultServerConfig,
	Transformer: DefaultTransformerConfig,
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Transformer TransformerConfig `yaml:"transformer"`
}

func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	cfg := DefaultConfig
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
