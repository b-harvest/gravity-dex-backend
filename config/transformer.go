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
	MongoDB:                  DefaultMongoDBConfig,
	Log:                      zap.NewProductionConfig(),
}

type TransformerConfig struct {
	BlockDataDir             string        `yaml:"block_data_dir"`
	BlockDataFilename        string        `yaml:"block_data_filename"`
	BlockDataWaitingInterval time.Duration `yaml:"block_data_waiting_interval"`
	PruningOffset            int           `yaml:"pruning_offset"`
	MongoDB                  MongoDBConfig `yaml:"mongodb"`
	Log                      zap.Config    `yaml:"log"`
}

func (cfg TransformerConfig) Validate() error {
	if cfg.BlockDataDir == "" {
		return fmt.Errorf("'block_data_dir' is required")
	}
	return nil
}
