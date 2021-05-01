package config

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

var DefaultTransformerConfig = TransformerConfig{
	BlockDataFilename:        "%08d/%d.json",
	BlockDataBucketSize:      10000,
	BlockDataWaitingInterval: time.Second,
	PruningOffset:            -2,
	MongoDB:                  DefaultMongoDBConfig,
	Log:                      zap.NewProductionConfig(),
}

type TransformerConfig struct {
	BlockDataDir             string        `yaml:"block_data_dir"`
	BlockDataFilename        string        `yaml:"block_data_filename"`
	BlockDataBucketSize      int           `yaml:"block_data_bucket_size"`
	BlockDataWaitingInterval time.Duration `yaml:"block_data_waiting_interval"`
	PruningOffset            int           `yaml:"pruning_offset"`
	IgnoredAddresses         []string      `yaml:"ignored_addresses"`
	MongoDB                  MongoDBConfig `yaml:"mongodb"`
	Log                      zap.Config    `yaml:"log"`
}

func (cfg TransformerConfig) Validate() error {
	if cfg.BlockDataDir == "" {
		return fmt.Errorf("'block_data_dir' is required")
	}
	return nil
}

func (cfg TransformerConfig) IgnoredAddressesSet() map[string]struct{} {
	s := make(map[string]struct{})
	for _, a := range cfg.IgnoredAddresses {
		s[a] = struct{}{}
	}
	return s
}
