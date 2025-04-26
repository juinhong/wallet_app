package services

import (
	"Crypto.com/internal/repositories/postgres"
	"context"
	"errors"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"

	"Crypto.com/internal/models"
	"Crypto.com/mocks"
	"github.com/golang/mock/gomock"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestWalletService_Deposit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWalletRepository(ctrl)
	mockCache := mocks.NewMockCacheRepository(ctrl)
	logger := logrus.New()
	service := NewWalletService(mockRepo, mockCache, logger)

	t.Run("successful deposit", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().Deposit(ctx, "user1", 100.0).Return(nil)
		mockCache.EXPECT().InvalidateBalance(gomock.Any(), "user1").Return(nil)

		err := service.Deposit(ctx, "user1", 100.0)
		assert.NoError(t, err)
	})

	t.Run("invalid amount", func(t *testing.T) {
		err := service.Deposit(context.Background(), "user1", -50.0)
		assert.ErrorIs(t, err, postgres.ErrInvalidAmount)
	})

	t.Run("repository error", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().Deposit(ctx, "user1", 100.0).Return(errors.New("db error"))

		err := service.Deposit(ctx, "user1", 100.0)
		assert.ErrorContains(t, err, "db error")
	})
}

func TestWalletService_Withdraw(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWalletRepository(ctrl)
	mockCache := mocks.NewMockCacheRepository(ctrl)
	service := NewWalletService(mockRepo, mockCache, logrus.New())

	t.Run("successful withdrawal", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().Withdraw(ctx, "user1", 50.0).Return(nil)
		mockCache.EXPECT().InvalidateBalance(ctx, "user1").Return(nil)

		err := service.Withdraw(ctx, "user1", 50.0)
		assert.NoError(t, err)
	})

	t.Run("insufficient funds", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().Withdraw(ctx, "user1", 100.0).Return(postgres.ErrInsufficientBalance)

		err := service.Withdraw(ctx, "user1", 100.0)
		assert.ErrorIs(t, err, postgres.ErrInsufficientBalance)
	})
}

func TestWalletService_Transfer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWalletRepository(ctrl)
	mockCache := mocks.NewMockCacheRepository(ctrl)
	service := NewWalletService(mockRepo, mockCache, logrus.New())

	t.Run("successful transfer", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().Transfer(ctx, "user1", "user2", 75.0).Return(nil)
		mockCache.EXPECT().InvalidateBalance(ctx, "user1").Return(nil)
		mockCache.EXPECT().InvalidateBalance(ctx, "user2").Return(nil)

		err := service.Transfer(ctx, "user1", "user2", 75.0)
		assert.NoError(t, err)
	})

	t.Run("same user transfer", func(t *testing.T) {
		err := service.Transfer(context.Background(), "user1", "user1", 10.0)
		assert.ErrorIs(t, err, postgres.ErrInvalidUserID)
	})

	t.Run("invalid amount", func(t *testing.T) {
		err := service.Transfer(context.Background(), "user1", "user2", -5.0)
		assert.ErrorIs(t, err, postgres.ErrInvalidAmount)
	})
}

func TestWalletService_GetBalance(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWalletRepository(ctrl)
	mockCache := mocks.NewMockCacheRepository(ctrl)
	service := NewWalletService(mockRepo, mockCache, logrus.New())

	t.Run("cache hit", func(t *testing.T) {
		ctx := context.Background()
		mockCache.EXPECT().GetBalance(ctx, "user1").Return(150.0, nil)

		balance, err := service.GetBalance(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 150.0, balance)
	})

	t.Run("cache miss", func(t *testing.T) {
		ctx := context.Background()
		mockCache.EXPECT().GetBalance(ctx, "user1").Return(0.0, goredis.Nil)
		mockRepo.EXPECT().GetBalance(ctx, "user1").Return(200.0, nil)
		mockCache.EXPECT().SetBalance(gomock.Any(), "user1", 200.0).Return(nil)

		balance, err := service.GetBalance(ctx, "user1")
		assert.NoError(t, err)
		assert.Equal(t, 200.0, balance)
	})
}

func TestWalletService_GetTransactionHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockWalletRepository(ctrl)
	service := NewWalletService(mockRepo, nil, logrus.New())

	t.Run("default limit", func(t *testing.T) {
		ctx := context.Background()
		ct := time.Now()
		expected := []models.Transaction{{CreatedAt: &ct, Amount: proto.Float64(100.0)}}
		mockRepo.EXPECT().GetTransactionHistory(ctx, "user1", 50, 0).Return(expected, nil)

		result, err := service.GetTransactionHistory(ctx, "user1", 0, 0)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})

	t.Run("custom limit", func(t *testing.T) {
		ctx := context.Background()
		mockRepo.EXPECT().GetTransactionHistory(ctx, "user1", 75, 10).Return(nil, nil)

		_, err := service.GetTransactionHistory(ctx, "user1", 75, 10)
		assert.NoError(t, err)
	})
}
