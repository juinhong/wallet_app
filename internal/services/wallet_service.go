package services

import (
	"context"

	"github.com/sirupsen/logrus"

	"Crypto.com/internal/models"
	"Crypto.com/internal/repositories/postgres"
	"Crypto.com/internal/repositories/redis"
)

type WalletService struct {
	repo   postgres.WalletRepository
	cache  redis.CacheRepository
	logger *logrus.Logger
}

func NewWalletService(repo postgres.WalletRepository, cache redis.CacheRepository, logger *logrus.Logger) *WalletService {
	return &WalletService{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (s *WalletService) Deposit(ctx context.Context, userID string, amount float64) error {
	s.logger.WithFields(logrus.Fields{
		"userID": userID,
		"amount": amount,
	}).Debug("Processing deposit")

	err := s.repo.Deposit(ctx, userID, amount)
	if err == nil {
		_ = s.cache.InvalidateBalance(ctx, userID)
	}
	return err
}

func (s *WalletService) Withdraw(ctx context.Context, userID string, amount float64) error {
	err := s.repo.Withdraw(ctx, userID, amount)
	if err == nil {
		_ = s.cache.InvalidateBalance(ctx, userID)
	}
	return err
}

func (s *WalletService) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	err := s.repo.Transfer(ctx, fromUserID, toUserID, amount)
	if err == nil {
		// Invalidate both accounts
		_ = s.cache.InvalidateBalance(ctx, fromUserID)
		_ = s.cache.InvalidateBalance(ctx, toUserID)
	}
	return err
}

func (s *WalletService) GetBalance(ctx context.Context, userID string) (float64, error) {
	// Check cache first
	if balance, err := s.cache.GetBalance(ctx, userID); err == nil {
		return balance, nil
	}

	// Fallback to database
	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		return 0, err
	}

	// Update cache
	go func() {
		_ = s.cache.SetBalance(ctx, userID, balance)
	}()

	return balance, nil
}

func (s *WalletService) GetTransactionHistory(ctx context.Context, userID string, limit, offset int) ([]models.Transaction, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetTransactionHistory(ctx, userID, limit, offset)
}
