package schema

import (
	"time"
)

const (
	CheckpointBlockHeightKey = "blockHeight"
	CheckpointTimestampKey   = "timestamp"
)

type Checkpoint struct {
	BlockHeight int64     `bson:"blockHeight"`
	Timestamp   time.Time `bson:"timestamp"`
}

const (
	AccountBlockHeightKey = "blockHeight"
	AccountUsernameKey    = "username"
	AccountAddressKey     = "address"
	AccountCoinsKey       = "coins"
	AccountActionsKey     = "actions"
)

type Account struct {
	BlockHeight int64                  `bson:"blockHeight"`
	Username    string                 `bson:"username"`
	Address     string                 `bson:"address"`
	Coins       []Coin                 `bson:"coins"`
	Actions     map[string]ActionState `bson:"actions"`
}

type Coin struct {
	Denom  string `bson:"denom"`
	Amount int64  `bson:"amount"`
}

type ActionState struct {
	Deposits []DepositAction `bson:"deposits"`
	Swaps    []SwapAction    `bson:"swaps"`
}

type SwapAction struct {
	Timestamp time.Time `bson:"timestamp"`
}

type DepositAction struct {
	Timestamp time.Time `bson:"timestamp"`
}

const (
	SupplyBlockHeightKey = "blockHeight"
)

type Supply struct {
	BlockHeight int64 `bson:"blockHeight"`
	Coin
}

const (
	PoolBlockHeightKey = "blockHeight"
)

type Pool struct {
	BlockHeight    int64  `bson:"blockHeight"`
	ReserveCoins   []Coin `bson:"reserveCoins"`
	PoolCoinDenom  string `bson:"poolCoinDenom"`
	PoolCoinSupply int64  `bson:"poolCoinSupply"`
}
