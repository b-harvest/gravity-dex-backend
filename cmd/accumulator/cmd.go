package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	var redisURL string
	var blockDataDir string
	var numWorkers int
	cmd := &cobra.Command{
		Use: "accumulator",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			rp := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.DialURL(redisURL)
				},
			}

			acc, err := NewAccumulator(blockDataDir)
			if err != nil {
				return fmt.Errorf("new accumulator: %w", err)
			}

			cm := NewCacheManager(rp, CacheKey)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c, err := cm.Get(ctx)
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
				st, err = acc.Run(ctx, st, blockHeight, h, numWorkers)
				if err != nil {
					return fmt.Errorf("run accumulator: %w", err)
				}
				log.Printf("accumulated state in %v", time.Since(started))

				if err := cm.Set(ctx, &Cache{
					BlockHeight: h,
					Stats:       st,
				}); err != nil {
					return fmt.Errorf("set cache: %w", err)
				}
			}

			log.Printf("active addresses = %d", len(st.ActiveAddresses))
			log.Printf("extracting last 24h information")
			now := time.Now().UTC().Truncate(time.Hour)
			past := now.AddDate(0, 0, -1)
			for !past.After(now) {
				hourKey := HourKey(past)
				hs, ok := st.ByHour[hourKey]
				if ok {
					ts := 0
					for _, n := range hs.NumDepositsByPoolID {
						ts += n
					}
					for _, n := range hs.NumSwapsByPoolID {
						ts += n
					}
					log.Printf("[%s] total deposit/swaps = %d", hourKey, ts)
				}
				past = past.Add(time.Hour)
			}

			return nil
		},
	}
	cmd.Flags().StringVarP(&redisURL, "redis", "r", "redis://localhost", "redis url")
	cmd.Flags().StringVarP(&blockDataDir, "dir", "d", "", "block data dir")
	cmd.Flags().IntVarP(&numWorkers, "workers", "n", runtime.NumCPU(), "number of concurrent workers")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}
