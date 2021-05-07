package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	var redisURL string
	var blockDataDir string
	var numWorkers int
	var updateInterval time.Duration
	var bindAddr string
	cmd := &cobra.Command{
		Use: "accumulator",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			rp := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.DialURL(redisURL)
				},
			}

			cm := NewCacheManager(rp, CacheKey)
			acc, err := NewAccumulator(blockDataDir, cm)
			if err != nil {
				return fmt.Errorf("new accumulator: %w", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			s := NewServer(cm)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					if err := acc.Run(ctx, numWorkers); err != nil {
						log.Printf("failed to run accumulator: %v", err)
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(updateInterval):
					}
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				log.Printf("server started on %s", bindAddr)
				if err := s.Start(bindAddr); err != nil {
					log.Fatalf("failed run server: %v", err)
				}
			}()

			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
			<-sigs
			signal.Reset(syscall.SIGINT, syscall.SIGTERM)

			log.Printf("gracefully shutting down")
			cancel()
			if err := s.ShutdownWithTimeout(10 * time.Second); err != nil {
				log.Printf("failed to shutdown server: %v", err)
			}
			wg.Wait()

			return nil
		},
	}
	cmd.Flags().StringVarP(&redisURL, "redis", "r", "redis://localhost", "redis url")
	cmd.Flags().StringVarP(&blockDataDir, "dir", "d", "", "block data dir")
	cmd.Flags().IntVarP(&numWorkers, "workers", "n", runtime.NumCPU(), "number of concurrent workers")
	cmd.Flags().DurationVarP(&updateInterval, "interval", "i", 30*time.Second, "update interval")
	cmd.Flags().StringVarP(&bindAddr, "bind", "b", "0.0.0.0:9000", "binding address")
	_ = cmd.MarkFlagRequired("dir")
	return cmd
}
