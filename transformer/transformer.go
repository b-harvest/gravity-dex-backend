package transformer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/util"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Transformer struct {
	cfg    config.TransformerConfig
	mc     *mongo.Client
	logger *zap.Logger
}

func New(cfg config.TransformerConfig, mc *mongo.Client, logger *zap.Logger) (*Transformer, error) {
	return &Transformer{cfg, mc, logger}, nil
}

func (t *Transformer) CheckpointCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.CheckpointCollection)
}

func (t *Transformer) AccountCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.AccountCollection)
}

func (t *Transformer) PoolCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.PoolCollection)
}

func (t *Transformer) Run(ctx context.Context) error {
	for {
		t.logger.Debug("getting latest block height")
		h, err := t.LatestBlockHeight(ctx)
		if err != nil {
			return fmt.Errorf("get latest block height: %w", err)
		}
		t.logger.Debug("got latest block height", zap.Int64("height", h))
		if h > 1 {
			t.logger.Debug("pruning old state", zap.Int64("height", h))
			if err := t.PruneOldState(ctx, h+int64(t.cfg.PruningOffset)); err != nil {
				return fmt.Errorf("prune old state: %w", err)
			}
		}
		updates, data, err := t.AccStateUpdates(ctx, h+1)
		if err != nil {
			return fmt.Errorf("accumulate state updates: %w", err)
		}
		lastH := data.Header.Height
		t.logger.Info("updating state", zap.Int64("from", h+1), zap.Int64("to", lastH))
		if err := t.UpdateState(ctx, h, lastH, updates, data); err != nil {
			return fmt.Errorf("update state: %w", err)
		}
		t.logger.Debug("updating latest block height", zap.Int64("height", lastH))
		if err := t.UpdateLatestBlockHeight(ctx, lastH); err != nil {
			return fmt.Errorf("update latest block height: %w", err)
		}
	}
}

func (t *Transformer) LatestBlockHeight(ctx context.Context) (int64, error) {
	var cp schema.Checkpoint
	if err := t.CheckpointCollection().FindOne(ctx, bson.M{
		schema.CheckpointBlockHeightKey: bson.M{"$exists": true},
	}).Decode(&cp); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	return cp.BlockHeight, nil
}

func (t *Transformer) UpdateLatestBlockHeight(ctx context.Context, height int64) error {
	if _, err := t.CheckpointCollection().UpdateOne(ctx, bson.M{
		schema.CheckpointBlockHeightKey: bson.M{"$exists": true},
	}, bson.M{
		"$set": bson.M{
			schema.CheckpointBlockHeightKey: height,
			schema.CheckpointTimestampKey:   time.Now(),
		},
	}, options.Update().SetUpsert(true)); err != nil {
		return err
	}
	return nil
}

func (t *Transformer) PruneOldState(ctx context.Context, currentBlockHeight int64) error {
	if _, err := t.AccountCollection().DeleteMany(ctx, bson.M{
		schema.AccountBlockHeightKey: bson.M{"$lt": currentBlockHeight},
	}); err != nil {
		return fmt.Errorf("prune account collection: %w", err)
	}
	if _, err := t.PoolCollection().DeleteMany(ctx, bson.M{
		schema.PoolBlockHeightKey: bson.M{"$lt": currentBlockHeight},
	}); err != nil {
		return fmt.Errorf("prune pool collection: %w", err)
	}
	return nil
}

func (t *Transformer) blockDataFilename(blockHeight int64) string {
	bs := int64(t.cfg.BlockDataBucketSize)
	p := blockHeight / bs * bs
	return filepath.Join(t.cfg.BlockDataDir, fmt.Sprintf(t.cfg.BlockDataFilename, p, blockHeight))
}

func (t *Transformer) ReadBlockData(blockHeight int64) (*BlockData, error) {
	f, err := os.Open(t.blockDataFilename(blockHeight))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data BlockData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (t *Transformer) WaitForBlockData(ctx context.Context, blockHeight int64) (*BlockData, error) {
	ticker := util.NewImmediateTicker(t.cfg.BlockDataWaitingInterval)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		t.logger.Debug("waiting for the block data", zap.Int64("height", blockHeight))
		data, err := t.ReadBlockData(blockHeight)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read block data: %w", err)
			}
		} else {
			return data, nil
		}
	}
}

func (t *Transformer) UpdateState(ctx context.Context, currentBlockHeight, lastBlockHeight int64, updates *StateUpdates, data *BlockData) error {
	balancesByAddr := data.BalancesByAddress()
	eg, ctx2 := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if err := t.UpdateAccounts(ctx2, currentBlockHeight, lastBlockHeight, updates, data, balancesByAddr); err != nil {
			return fmt.Errorf("update accounts: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := t.UpdatePools(ctx2, currentBlockHeight, lastBlockHeight, updates, data, balancesByAddr); err != nil {
			return fmt.Errorf("update pools: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

func (t *Transformer) UpdateAccounts(ctx context.Context, currentBlockHeight, lastBlockHeight int64, updates *StateUpdates, data *BlockData, balancesByAddr map[string][]schema.Coin) error {
	reserveAccAddrs := make(map[string]struct{})
	for _, p := range data.Pools {
		reserveAccAddrs[p.ReserveAccountAddress] = struct{}{}
	}
	var writes []mongo.WriteModel
	for addr, b := range balancesByAddr {
		if _, ok := reserveAccAddrs[addr]; ok {
			continue
		}
		var acc schema.Account
		if currentBlockHeight > 0 {
			if err := t.AccountCollection().FindOne(ctx, bson.M{
				schema.AccountBlockHeightKey: currentBlockHeight,
				schema.AccountAddressKey:     addr,
			}).Decode(&acc); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("find account: %w", err)
			}
		}
		acc.DepositStatus = schema.MergeAccountActionStatuses(acc.DepositStatus, updates.depositStatusByAddress[addr])
		acc.SwapStatus = schema.MergeAccountActionStatuses(acc.SwapStatus, updates.swapStatusByAddress[addr])
		writes = append(writes,
			mongo.NewUpdateOneModel().
				SetFilter(bson.M{
					schema.AccountBlockHeightKey: lastBlockHeight,
					schema.AccountAddressKey:     addr,
				}).
				SetUpdate(bson.M{
					"$set": bson.M{
						schema.AccountCoinsKey:         b,
						schema.AccountDepositStatusKey: acc.DepositStatus,
						schema.AccountSwapStatusKey:    acc.SwapStatus,
					},
					"$setOnInsert": bson.M{
						schema.AccountUsernameKey: acc.Username,
					},
				}).
				SetUpsert(true))
	}
	if len(writes) > 0 {
		if _, err := t.AccountCollection().BulkWrite(ctx, writes); err != nil {
			return fmt.Errorf("bulk write: %w", err)
		}
	}
	return nil
}

func (t *Transformer) UpdatePools(ctx context.Context, currentBlockHeight, lastBlockHeight int64, updates *StateUpdates, data *BlockData, balancesByAddr map[string][]schema.Coin) error {
	var writes []mongo.WriteModel
	for _, p := range data.Pools {
		var reserveCoins []schema.Coin
		for _, d := range p.ReserveCoinDenoms {
			var amount int64
			for _, c := range balancesByAddr[p.ReserveAccountAddress] {
				if c.Denom == d {
					amount = c.Amount
					break
				}
			}
			reserveCoins = append(reserveCoins, schema.Coin{Denom: d, Amount: amount})
		}
		// this is not necessary if it assumed that reserve coin denoms in data are already sorted.
		sort.Slice(reserveCoins, func(i, j int) bool { return reserveCoins[i].Denom < reserveCoins[j].Denom })
		var pool schema.Pool
		if currentBlockHeight > 0 {
			if err := t.PoolCollection().FindOne(ctx, bson.M{
				schema.PoolBlockHeightKey: currentBlockHeight,
				schema.PoolIDKey:          p.Id,
			}).Decode(&pool); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
				return fmt.Errorf("find pool: %w", err)
			}
		}
		pool.SwapFeeVolumes = schema.MergeVolumes(pool.SwapFeeVolumes, updates.swapVolumesByPoolID[p.Id])
		pool.SwapFeeVolumes.RemoveOutdated(data.Header.Time.Add(-time.Hour))
		writes = append(writes,
			mongo.NewUpdateOneModel().
				SetFilter(bson.M{
					schema.PoolBlockHeightKey: lastBlockHeight,
					schema.PoolIDKey:          p.Id,
				}).
				SetUpdate(bson.M{
					"$set": bson.M{
						schema.PoolReserveCoins: reserveCoins,
						schema.PoolPoolCoinKey: schema.Coin{
							Denom:  p.PoolCoinDenom,
							Amount: data.BankModuleState.Supply.AmountOf(p.PoolCoinDenom).Int64(),
						},
						schema.PoolSwapFeeVolumesKey: pool.SwapFeeVolumes,
					},
				}).
				SetUpsert(true))
	}
	if len(writes) > 0 {
		if _, err := t.PoolCollection().BulkWrite(ctx, writes); err != nil {
			return fmt.Errorf("bulk write: %w", err)
		}
	}
	return nil
}
