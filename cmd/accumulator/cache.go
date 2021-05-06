package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/gomodule/redigo/redis"
	jsoniter "github.com/json-iterator/go"
)

var CacheKey = "gdex-accumulator:cache"

type Cache struct {
	BlockHeight int64
	Stats       *Stats
}

type CacheManager struct {
	rp  *redis.Pool
	key string
}

func NewCacheManager(rp *redis.Pool, key string) *CacheManager {
	return &CacheManager{rp: rp, key: key}
}

func (cm *CacheManager) Get(ctx context.Context) (*Cache, error) {
	conn, err := cm.rp.GetContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get redis conn: %w", err)
	}
	defer conn.Close()
	b, err := redis.Bytes(conn.Do("GET", cm.key))
	if err != nil {
		if errors.Is(err, redis.ErrNil) {
			return nil, nil
		}
		return nil, err
	}
	var c Cache
	if err := jsoniter.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("unmarshal cache: %w", err)
	}
	return &c, nil
}

func (cm *CacheManager) Set(ctx context.Context, c *Cache) error {
	conn, err := cm.rp.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("get redis conn: %w", err)
	}
	defer conn.Close()
	b, err := jsoniter.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	if _, err := conn.Do("SET", cm.key, b); err != nil {
		return err
	}
	return nil
}
