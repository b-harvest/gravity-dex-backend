package transformer

import (
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/b-harvest/gravity-dex-backend/schema"
)

type BlockData struct {
	Header          tmproto.Header          `json:"block_header"`
	BankModuleState *banktypes.GenesisState `json:"bank_module_states"`
	Events          []abcitypes.Event       `json:"end_block_events"`
	Pools           []liquiditytypes.Pool   `json:"pools"`
}

func (d *BlockData) BalancesByAddress() map[string][]schema.Coin {
	m := make(map[string][]schema.Coin)
	if d.BankModuleState != nil {
		for _, b := range d.BankModuleState.Balances {
			coins := []schema.Coin{}
			for _, c := range b.Coins {
				coins = append(coins, schema.Coin{Denom: c.Denom, Amount: c.Amount.Int64()})
			}
			m[b.Address] = coins
		}
	}
	return m
}

func (d *BlockData) PoolByID() map[uint64]liquiditytypes.Pool {
	m := make(map[uint64]liquiditytypes.Pool)
	for _, p := range d.Pools {
		m[p.Id] = p
	}
	return m
}

func oppositeReserveCoinDenom(pool liquiditytypes.Pool, denom string) (string, bool) {
	for _, d := range pool.ReserveCoinDenoms {
		if d != denom {
			return d, true
		}
	}
	return "", false
}
