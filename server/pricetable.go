package server

import "fmt"

var (
	_ Coin = StableCoin{}
	_ Coin = StakingCoin{}
	_ Coin = PoolCoin{}
)

type Coin interface {
	Amount() int64
	Denom() string
}

type StableCoin struct {
	amount int64
	denom  string
}

func NewStableCoin(amount int64, denom string) Coin {
	return StableCoin{amount, denom}
}

func (c StableCoin) Amount() int64 {
	return c.amount
}

func (c StableCoin) Denom() string {
	return c.denom
}

type StakingCoin struct {
	amount int64
	denom  string
}

func NewStakingCoin(amount int64, denom string) Coin {
	return StakingCoin{amount, denom}
}

func (c StakingCoin) Amount() int64 {
	return c.amount
}

func (c StakingCoin) Denom() string {
	return c.denom
}

type PoolCoin struct {
	amount int64
	pool   *Pool
}

func NewPoolCoin(amount int64, pool *Pool) Coin {
	return PoolCoin{amount, pool}
}

func (c PoolCoin) Amount() int64 {
	return c.amount
}

func (c PoolCoin) Denom() string {
	return c.pool.PoolCoinDenom
}

func (c PoolCoin) Share() float64 {
	return float64(c.amount) / float64(c.pool.PoolCoinSupply)
}

type Pool struct {
	ReserveCoins   []Coin
	PoolCoinDenom  string
	PoolCoinSupply int64
}

type PriceTable map[string]float64

func (pt PriceTable) Value(coin Coin) (float64, error) {
	p, ok := pt[coin.Denom()]
	if !ok {
		switch c := coin.(type) {
		case StableCoin:
			p = 1
		case StakingCoin:
			return 0, fmt.Errorf("staking coin %q's price must be in PriceTable", coin.Denom())
		case PoolCoin:
			sum := 0.0
			for _, rc := range c.pool.ReserveCoins {
				t, err := pt.Value(rc)
				if err != nil {
					return 0, err
				}
				sum += t
			}
			p = 1 / float64(c.pool.PoolCoinSupply) * sum
		default:
			return 0, fmt.Errorf("wrong coin type: %T", c)
		}
		pt[coin.Denom()] = p
	}
	return p * float64(coin.Amount()), nil
}
