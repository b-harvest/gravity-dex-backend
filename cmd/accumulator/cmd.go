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

			yesterday := time.Now().AddDate(0, 0, -1)
			log.Printf("active addresses = %d", len(st.ActiveAddresses))

			log.Printf("total %d deposits, %d swaps", st.NumDeposits(), st.NumSwaps())
			log.Printf("(last 24 hours) %d deposits, %d swaps", st.NumDepositsSince(yesterday), st.NumSwapsSince(yesterday))

			v := st.OfferCoins()
			v.Add(st.DemandCoins())
			log.Printf("total swapped coins (offer coins + demand coins) = %s", v)
			v = st.OfferCoinsSince(yesterday)
			v.Add(st.DemandCoinsSince(yesterday))
			log.Printf("(last 24 hours) total swapped coins (offer coins + demand coins) = %s", v)

			log.Printf("hint: you can obtain the swap volume by first calculating " +
				"the value of swapped coins(lookup the price table!) then " +
				"divide it by 2(since it is the sum of offer coins AND demand coins)")

			return nil
		},
	}
	cmd.Flags().StringVarP(&redisURL, "redis", "r", "redis://localhost", "redis url")
	cmd.Flags().StringVarP(&blockDataDir, "dir", "d", "", "block data dir")
	cmd.Flags().IntVarP(&numWorkers, "workers", "n", runtime.NumCPU(), "number of concurrent workers")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}
