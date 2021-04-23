package config

import (
	"fmt"

	"go.uber.org/zap"
)

var DefaultServerConfig = ServerConfig{
	BindAddr: "0.0.0.0:8080",
	DenomMetadata: map[string]DenomMetadata{
		"uatom": {"atom", 6},
		"uxrn":  {"xrn", 6},
		"ubtsg": {"btsg", 6},
		"udvpn": {"dvpn", 6},
		"uxprt": {"xprt", 6},
		"uakt":  {"akt", 6},
		"uluna": {"luna", 6},
		"ungm":  {"ngm", 6},
		"ugcyb": {"gcyb", 6},
		"uiris": {"iris", 6},
		"xrun":  {"run", 6},
		"uusd":  {"usd", 6},
	},
	MongoDB: DefaultMongoDBConfig,
	Log:     zap.NewProductionConfig(),
}

type ServerConfig struct {
	BindAddr            string                   `yaml:"bind_addr"`
	StableCoinDenoms    []string                 `yaml:"stable_coin_denoms"`
	StakingCoinDenoms   []string                 `yaml:"staking_coin_denoms"`
	DenomMetadata       map[string]DenomMetadata `yaml:"denom_metadata"`
	CoinMarketCapAPIKey string                   `yaml:"cmc_api_key"`
	MongoDB             MongoDBConfig            `yaml:"mongodb"`
	Log                 zap.Config               `yaml:"log"`
}

func (cfg ServerConfig) Validate() error {
	if len(cfg.StableCoinDenoms) == 0 {
		return fmt.Errorf("'stable_coin_denoms' is empty")
	}
	if len(cfg.StakingCoinDenoms) == 0 {
		return fmt.Errorf("'staking_coin_denoms' is empty")
	}
	if cfg.CoinMarketCapAPIKey == "" {
		return fmt.Errorf("'cmc_api_key' is required")
	}
	return nil
}

type DenomMetadata struct {
	Display  string `yaml:"display"`
	Exponent int    `yaml:"exponent"`
}
