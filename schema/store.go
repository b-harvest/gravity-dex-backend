package schema

import "time"

const (
	CheckpointBlockHeightKey = "blockHeight"
	CheckpointTimestampKey   = "timestamp"
)

type Checkpoint struct {
	BlockHeight int64     `bson:"blockHeight"`
	Timestamp   time.Time `bson:"timestamp"`
}

const (
	AccountBlockHeightKey   = "blockHeight"
	AccountUsernameKey      = "username"
	AccountAddressKey       = "address"
	AccountCoinsKey         = "coins"
	AccountDepositStatusKey = "depositStatus"
	AccountSwapStatusKey    = "swapStatus"
)

type Account struct {
	BlockHeight   int64               `bson:"blockHeight"`
	Username      string              `bson:"username"`
	Address       string              `bson:"address"`
	Coins         []Coin              `bson:"coins"`
	DepositStatus AccountActionStatus `bson:"depositStatus"`
	SwapStatus    AccountActionStatus `bson:"swapStatus"`
}

type Coin struct {
	Denom  string `bson:"denom"`
	Amount int64  `bson:"amount"`
}

type AccountActionStatus struct {
	CountByPoolID map[uint64]int `bson:"countByPoolID"`
	CountByDate   map[string]int `bson:"countByDate"`
}

func NewAccountActionStatus() AccountActionStatus {
	return AccountActionStatus{
		CountByPoolID: make(map[uint64]int),
		CountByDate:   make(map[string]int),
	}
}

func MergeAccountActionStatuses(ss ...AccountActionStatus) AccountActionStatus {
	s := AccountActionStatus{
		CountByPoolID: make(map[uint64]int),
		CountByDate:   make(map[string]int),
	}
	for _, s2 := range ss {
		for id, c := range s2.CountByPoolID {
			s.CountByPoolID[id] += c
		}
		for date, c := range s2.CountByDate {
			s.CountByDate[date] += c
		}
	}
	return s
}

const (
	PoolBlockHeightKey    = "blockHeight"
	PoolIDKey             = "id"
	PoolReserveCoins      = "reserveCoins"
	PoolPoolCoinKey       = "poolCoin"
	PoolSwapFeeVolumesKey = "swapFeeVolumes"
)

type Pool struct {
	BlockHeight    int64   `bson:"blockHeight"`
	ID             uint64  `bson:"id"`
	ReserveCoins   []Coin  `bson:"reserveCoins"`
	PoolCoin       Coin    `bson:"poolCoin"`
	SwapFeeVolumes Volumes `bson:"swapFeeVolumes"`
}

const VolumeTimeUnit = time.Minute

type Volumes map[int64]CoinMap

func MergeVolumes(vs ...Volumes) Volumes {
	v := make(Volumes)
	for _, v2 := range vs {
		for t, c2 := range v2 {
			t = time.Unix(t, 0).Truncate(VolumeTimeUnit).Unix()
			c, ok := v[t]
			if !ok {
				c = make(CoinMap)
				v[t] = c
			}
			c.Add(c2)
		}
	}
	return v
}

func (v Volumes) TotalCoins() CoinMap {
	c := make(CoinMap)
	for _, c2 := range v {
		c.Add(c2)
	}
	return c
}

func (v Volumes) AddCoins(now time.Time, c2 CoinMap) {
	t := now.Truncate(VolumeTimeUnit).Unix()
	c, ok := v[t]
	if !ok {
		c = make(CoinMap)
		v[t] = c
	}
	c.Add(c2)
}

func (v Volumes) RemoveOutdated(past time.Time) {
	p := past.UTC().Unix()
	for t := range v {
		if t < p {
			delete(v, t)
		}
	}
}

type CoinMap map[string]int64

func (c CoinMap) Add(c2 CoinMap) {
	for denom, amount := range c2 {
		c[denom] += amount
	}
}

const (
	EventVisibleAtKey = "visibleAt"
	EventStartsAtKey  = "startsAt"
	EventEndsAtKey    = "endsAt"
)

type Event struct {
	Text      string    `bson:"text"`
	URL       string    `bson:"url"`
	VisibleAt time.Time `bson:"visibleAt"`
	StartsAt  time.Time `bson:"startsAt"`
	EndsAt    time.Time `bson:"endsAt"`
}
