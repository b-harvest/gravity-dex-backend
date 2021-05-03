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
	AccountAddressKey       = "address"
	AccountCoinsKey         = "coins"
	AccountDepositStatusKey = "depositStatus"
	AccountSwapStatusKey    = "swapStatus"
	AccountMetadataKey      = "metadata"
)

type Account struct {
	BlockHeight   int64               `bson:"blockHeight"`
	Address       string              `bson:"address"`
	Coins         []Coin              `bson:"coins"`
	DepositStatus AccountActionStatus `bson:"depositStatus"`
	SwapStatus    AccountActionStatus `bson:"swapStatus"`
	Metadata      *AccountMetadata    `bson:"metadata"`
}

type Coin struct {
	Denom  string `bson:"denom"`
	Amount int64  `bson:"amount"`
}

type AccountActionStatus struct {
	CountByPoolID       CountByPoolID            `bson:"countByPoolID"`
	CountByPoolIDByDate map[string]CountByPoolID `bson:"countByPoolIDByDate"`
}

type CountByPoolID map[uint64]int

func NewAccountActionStatus() AccountActionStatus {
	return AccountActionStatus{
		CountByPoolID:       make(CountByPoolID),
		CountByPoolIDByDate: make(map[string]CountByPoolID),
	}
}

func (s AccountActionStatus) NumDifferentPools() int {
	return len(s.CountByPoolID)
}

func (s AccountActionStatus) NumDifferentPoolsByDate() map[string]int {
	m := make(map[string]int)
	for date, c := range s.CountByPoolIDByDate {
		m[date] = len(c)
	}
	return m
}

func (s *AccountActionStatus) IncreaseCount(poolID uint64, date string, amount int) {
	s.CountByPoolID[poolID] += amount
	c, ok := s.CountByPoolIDByDate[date]
	if !ok {
		c = make(CountByPoolID)
		s.CountByPoolIDByDate[date] = c
	}
	c[poolID] += amount
}

func MergeAccountActionStatuses(ss ...AccountActionStatus) AccountActionStatus {
	s := AccountActionStatus{
		CountByPoolID:       make(CountByPoolID),
		CountByPoolIDByDate: make(map[string]CountByPoolID),
	}
	for _, s2 := range ss {
		for date, c := range s2.CountByPoolIDByDate {
			for id, c2 := range c {
				s.IncreaseCount(id, date, c2)
			}
		}
	}
	return s
}

const (
	AccountMetadataAddressKey   = "address"
	AccountMetadataIsBlockedKey = "isBlocked"
)

type AccountMetadata struct {
	Address   string    `bson:"address"`
	Username  string    `bson:"username"`
	IsBlocked bool      `bson:"isBlocked"`
	BlockedAt time.Time `bson:"blockedAt"`
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
	BannerVisibleAtKey = "visibleAt"
	BannerStartsAtKey  = "startsAt"
	BannerEndsAtKey    = "endsAt"
)

type Banner struct {
	UpcomingText string    `bson:"upcomingText"`
	Text         string    `bson:"text"`
	URL          string    `bson:"url"`
	VisibleAt    time.Time `bson:"visibleAt"`
	StartsAt     time.Time `bson:"startsAt"`
	EndsAt       time.Time `bson:"endsAt"`
}
