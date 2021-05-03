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

var DefaultMongoDBConfig = MongoDBConfig{
	URI:                  "mongodb://localhost",
	DB:                   "gdex",
	CheckpointCollection: "checkpoint",
	AccountCollection:    "accounts",
	PoolCollection:       "pools",
	BannerCollection:     "banners",
}

type MongoDBConfig struct {
	URI                  string `yaml:"uri"`
	DB                   string `yaml:"db"`
	CheckpointCollection string `yaml:"checkpoint_collection"`
	AccountCollection    string `yaml:"account_collection"`
	PoolCollection       string `yaml:"pool_collection"`
	BannerCollection     string `yaml:"banner_collection"`
}

var DefaultRedisConfig = RedisConfig{
	URI:              "redis://localhost",
	AccountsCacheKey: "gdex:accounts",
	PoolsCacheKey:    "gdex:pools",
	PricesCacheKey:   "gdex:prices",
}

type RedisConfig struct {
	URI              string `yaml:"uri"`
	AccountsCacheKey string `yaml:"accounts_cache_key"`
	PoolsCacheKey    string `yaml:"pools_cache_key"`
	PricesCacheKey   string `yaml:"prices_cache_key"`
}
