package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetBalance(ctx context.Context, userID string) (float64, error)
	SetBalance(ctx context.Context, userID string, balance float64) error
	InvalidateBalance(ctx context.Context, userID string) error
}

// Implementation remains the same
type CacheRepositoryImpl struct {
	client redis.Cmdable
	ttl    time.Duration
	logger *log.Logger
}

func NewCacheRepository(client redis.Cmdable, ttl time.Duration, logger *log.Logger) *CacheRepositoryImpl {
	return &CacheRepositoryImpl{
		client: client,
		ttl:    ttl,
		logger: logger,
	}
}

func (r *CacheRepositoryImpl) GetBalance(ctx context.Context, userID string) (float64, error) {
	val, err := r.client.Get(ctx, balanceKey(userID)).Result()

	if errors.Is(err, redis.Nil) {
		r.logger.Printf("DEBUG: Cache miss for balance key: %s", balanceKey(userID))
		return 0, redis.Nil
	}

	if err != nil {
		return 0, err
	}

	var balance float64
	err = json.Unmarshal([]byte(val), &balance)
	return balance, err
}

func (r *CacheRepositoryImpl) SetBalance(ctx context.Context, userID string, balance float64) error {
	serialized, err := json.Marshal(balance)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, balanceKey(userID), serialized, r.ttl).Err()
}

func (r *CacheRepositoryImpl) InvalidateBalance(ctx context.Context, userID string) error {
	return r.client.Del(ctx, balanceKey(userID)).Err()
}

func balanceKey(userID string) string {
	return "balance:" + userID
}
