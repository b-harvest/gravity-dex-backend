package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	jsoniter "github.com/json-iterator/go"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"golang.org/x/sync/errgroup"
)

type Accumulator struct {
	blockDataDir string
	cm           *CacheManager
}

func NewAccumulator(blockDataDir string, cm *CacheManager) (*Accumulator, error) {
	if _, err := os.Stat(blockDataDir); err != nil {
		return nil, fmt.Errorf("check block data dir: %w", err)
	}
	return &Accumulator{blockDataDir: blockDataDir, cm: cm}, nil
}

func (acc *Accumulator) LatestBlockBucket() (int64, error) {
	es, err := os.ReadDir(acc.blockDataDir)
	if err != nil {
		return 0, fmt.Errorf("read dir: %w", err)
	}
	var buckets []int64
	for _, e := range es {
		if !e.IsDir() {
			continue
		}
		var n int64
		if _, err := fmt.Sscanf(e.Name(), "%08d", &n); err != nil {
			continue
		}
		buckets = append(buckets, n)
	}
	if len(buckets) == 0 {
		return 0, fmt.Errorf("no buckets")
	}
	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i] > buckets[j]
	})
	return buckets[0], nil
}

func (acc *Accumulator) LatestBlockHeight() (int64, error) {
	bucket, err := acc.LatestBlockBucket()
	if err != nil {
		return 0, fmt.Errorf("get latest block bucket: %w", err)
	}
	es, err := os.ReadDir(acc.BlockDataBucketDir(bucket))
	if err != nil {
		return 0, fmt.Errorf("read dir: %w", err)
	}
	var heights []int64
	for _, e := range es {
		if e.IsDir() {
			continue
		}
		var height int64
		if _, err := fmt.Sscanf(e.Name(), "%d.json", &height); err != nil {
			continue
		}
		heights = append(heights, height)
	}
	if len(heights) == 0 {
		return 0, fmt.Errorf("no blocks")
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] > heights[j]
	})
	return heights[0], nil
}

func (acc *Accumulator) BlockDataBucketDir(bucket int64) string {
	return filepath.Join(acc.blockDataDir, fmt.Sprintf("%08d", bucket))
}

func (acc *Accumulator) BlockDataFilename(height int64) string {
	bs := int64(10000)
	p := height / bs * bs
	return filepath.Join(acc.blockDataDir, fmt.Sprintf("%08d", p), fmt.Sprintf("%d.json", height))
}

func (acc *Accumulator) ReadBlockData(height int64) (*BlockData, error) {
	f, err := os.Open(acc.BlockDataFilename(height))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var blockData BlockData
	if err := jsoniter.NewDecoder(f).Decode(&blockData); err != nil {
		return nil, err
	}
	if blockData.Header.Height != height {
		return nil, fmt.Errorf("wrong block height; expected %d, got %d", height, blockData.Header.Height)
	}
	return &blockData, nil
}

func (acc *Accumulator) UpdateStats(ctx context.Context, blockData *BlockData, stats *Stats) error {
	stats.mux.Lock()
	defer stats.mux.Unlock()
	hourKey := HourKey(blockData.Header.Time)
	poolByID := blockData.PoolByID()
	for _, evt := range blockData.Events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		switch evt.Type {
		case liquiditytypes.EventTypeDepositToPool:
			evt, err := NewDepositEvent(evt)
			if err != nil {
				return fmt.Errorf("extract deposit event: %w", err)
			}
			stats.AddActiveAddress(hourKey, evt.DepositorAddress)
			stats.AddNumDeposits(hourKey, evt.PoolID, 1)
		case liquiditytypes.EventTypeSwapTransacted:
			evt, err := NewSwapEvent(evt, poolByID)
			if err != nil {
				return fmt.Errorf("extract swap event: %w", err)
			}
			stats.AddActiveAddress(hourKey, evt.SwapRequesterAddress)
			stats.AddNumSwaps(hourKey, evt.PoolID, 1)
			stats.AddOfferCoins(hourKey, evt.PoolID, Coins{
				evt.ExchangedOfferCoin.Denom: evt.ExchangedDemandCoin.Amount.Int64(),
			})
			stats.AddDemandCoins(hourKey, evt.PoolID, Coins{
				evt.ExchangedDemandCoin.Denom: evt.ExchangedDemandCoin.Amount.Int64(),
			})
		}
	}
	return nil
}

func (acc *Accumulator) Accumulate(ctx context.Context, stats *Stats, startHeight, endHeight int64, numWorkers int) (*Stats, error) {
	if stats == nil {
		stats = NewStats()
	}
	jobs := make(chan int64, endHeight-startHeight)

	worker := func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case height, ok := <-jobs:
				if !ok {
					return nil
				}
				blockData, err := acc.ReadBlockData(height)
				if err != nil {
					return err
				}
				if err := acc.UpdateStats(ctx, blockData, stats); err != nil {
					return err
				}
			}
		}
	}

	eg, ctx2 := errgroup.WithContext(ctx)
	for i := 0; i < numWorkers; i++ {
		eg.Go(func() error {
			return worker(ctx2)
		})
	}

	for height := startHeight; height <= endHeight; height++ {
		jobs <- height
	}
	close(jobs)

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return stats, nil
}

func (acc *Accumulator) Run(ctx context.Context, numWorkers int) error {
	c, err := acc.cm.Get(ctx)
	if err != nil {
		return fmt.Errorf("get cache: %w", err)
	}
	var st *Stats
	blockHeight := int64(1)
	if c != nil {
		st = c.Stats
		blockHeight = c.BlockHeight
	}

	if blockHeight > 1 {
		log.Printf("last cached block height: %v", c.BlockHeight)
	} else {
		log.Printf("no cache found")
	}

	h, err := acc.LatestBlockHeight()
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}

	if blockHeight >= h {
		log.Printf("the state is up to date")
	} else {
		log.Printf("accumulating from %d to %d", blockHeight, h)

		started := time.Now()
		st, err = acc.Accumulate(ctx, st, blockHeight, h, numWorkers)
		if err != nil {
			return fmt.Errorf("run accumulator: %w", err)
		}
		log.Printf("accumulated state in %v", time.Since(started))

		if err := acc.cm.Set(ctx, &Cache{
			BlockHeight: h,
			Stats:       st,
		}); err != nil {
			return fmt.Errorf("set cache: %w", err)
		}
		log.Printf("saved cache")
	}

	return nil
}
