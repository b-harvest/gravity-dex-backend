package config

import "go.uber.org/zap"

var DefaultServerConfig = ServerConfig{
	BindAddr: "0.0.0.0:8080",
	Log:      zap.NewProductionConfig(),
}

type ServerConfig struct {
	BindAddr string     `yaml:"bind_addr"`
	Log      zap.Config `yaml:"log"`
}

func (cfg ServerConfig) Validate() error {
	return nil
}
