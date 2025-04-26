package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetBalance(ctx context.Context, userID string) (float64, error)
	SetBalance(ctx context.Context, userID string, balance float64) error
	InvalidateBalance(ctx context.Context, userID string) error
}

var (
	ErrInvalidUserID = errors.New("invalid user ID")
	ErrInvalidAmount = errors.New("invalid amount")
)

type CacheRepositoryImpl struct {
	client redis.Cmdable
	ttl    time.Duration
	logger *logrus.Logger
}

func NewCacheRepository(client redis.Cmdable, ttl time.Duration, logger *logrus.Logger) *CacheRepositoryImpl {
	return &CacheRepositoryImpl{
		client: client,
		ttl:    ttl,
		logger: logger,
	}
}

func (r *CacheRepositoryImpl) GetBalance(ctx context.Context, userID string) (float64, error) {
	if userID == "" {
		r.logger.Warn("GetBalance - userID cannot be an empty string")
		return 0, ErrInvalidUserID
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
	})

	val, err := r.client.Get(ctx, balanceKey(userID)).Result()

	if errors.Is(err, redis.Nil) {
		logger.Warn(fmt.Printf("GetBalance - cache miss: key = %v", balanceKey(userID)))
		return 0, redis.Nil
	}

	if err != nil {
		logger.WithError(err).Error(fmt.Printf("GetBalance - get cache error: key = %v", balanceKey(userID)))
		return 0, err
	}

	var balance float64
	err = json.Unmarshal([]byte(val), &balance)
	if err != nil {
		logger.WithError(err).Error(fmt.Printf("GetBalance - unmarshal error: key = %v, balance = %v", balanceKey(userID), balance))
		return 0, err
	}

	return balance, nil
}

func (r *CacheRepositoryImpl) SetBalance(ctx context.Context, userID string, balance float64) error {
	if userID == "" {
		r.logger.Warn("SetBalance - userID cannot be an empty string")
		return ErrInvalidUserID
	}

	if balance <= 0 {
		r.logger.Warn("SetBalance - balance must be greater than zero")
		return ErrInvalidAmount
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
		"amount": balance,
	})

	serialized, err := json.Marshal(balance)
	if err != nil {
		logger.WithError(err).Error("SetBalance - marshal error")
		return err
	}

	err = r.client.Set(ctx, balanceKey(userID), serialized, r.ttl).Err()
	if err != nil {
		logger.WithError(err).Error(fmt.Printf("SetBalance - set cache error: key = %v", balanceKey(userID)))
		return err
	}

	return nil
}

func (r *CacheRepositoryImpl) InvalidateBalance(ctx context.Context, userID string) error {
	if userID == "" {
		r.logger.Warn("InvalidateBalance - userID cannot be an empty string")
		return ErrInvalidUserID
	}

	err := r.client.Del(ctx, balanceKey(userID)).Err()
	if err != nil {
		r.logger.WithError(err).Error(fmt.Printf("InvalidateBalance - delete cache error: key = %v", balanceKey(userID)))
		return err
	}

	return nil
}

func balanceKey(userID string) string {
	return "balance:" + userID
}
