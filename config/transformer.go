package config

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

var DefaultTransformerConfig = TransformerConfig{
	BlockDataFilename:        "%d.json",
	BlockDataWaitingInterval: 500 * time.Millisecond,
	PruningOffset:            -2,
	MongoDB: MongoDBConfig{
		URI:                  "mongodb://localhost",
		DB:                   "gdex",
		CheckpointCollection: "checkpoint",
		AccountCollection:    "accounts",
		PoolCollection:       "pools",
	},
	Log: zap.NewProductionConfig(),
}

type TransformerConfig struct {
	BlockDataDir             string        `yaml:"block_data_dir"`
	BlockDataFilename        string        `yaml:"block_data_filename"`
	BlockDataWaitingInterval time.Duration `yaml:"block_data_waiting_interval"`
	PruningOffset            int           `yaml:"pruning_offset"`
	MongoDB                  MongoDBConfig `yaml:"mongodb"`
	Log                      zap.Config    `yaml:"log"`
}

type MongoDBConfig struct {
	URI                  string `yaml:"uri"`
	DB                   string `yaml:"db"`
	CheckpointCollection string `yaml:"checkpoint_collection"`
	AccountCollection    string `yaml:"account_collection"`
	PoolCollection       string `yaml:"pool_collection"`
}

func (cfg TransformerConfig) Validate() error {
	if cfg.BlockDataDir == "" {
		return fmt.Errorf("'block_data_dir' is required")
	}
	return nil
}
