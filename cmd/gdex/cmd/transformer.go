package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/service/store"
	"github.com/b-harvest/gravity-dex-backend/transformer"
)

func TransformerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transformer",
		Short: "run transformer",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			cfg, err := config.Load("config.yml")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if err := cfg.Transformer.Validate(); err != nil {
				return fmt.Errorf("validate transformer config: %w", err)
			}

			logger, err := cfg.Transformer.Log.Build()
			if err != nil {
				return fmt.Errorf("build logger: %w", err)
			}
			defer logger.Sync()

			mc, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.Transformer.MongoDB.URI))
			if err != nil {
				return fmt.Errorf("connect mongodb: %w", err)
			}
			defer mc.Disconnect(context.Background())
			if err := mc.Ping(context.Background(), nil); err != nil {
				return fmt.Errorf("ping mongodb: %w", err)
			}

			t, err := transformer.New(cfg.Transformer, mc, logger)
			if err != nil {
				return fmt.Errorf("new transformer: %w", err)
			}

			ss := store.NewService(cfg.Transformer.MongoDB, mc)
			names, err := ss.EnsureDBIndexes(context.Background())
			if err != nil {
				return fmt.Errorf("ensure db indexes: %w", err)
			}
			logger.Info("created db indexes", zap.Strings("names", names))

			logger.Info("started")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan error)
			go func() {
				done <- t.Run(ctx)
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit

			logger.Info("gracefully shutting down")
			cancel()

			if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		},
	}
	return cmd
}
