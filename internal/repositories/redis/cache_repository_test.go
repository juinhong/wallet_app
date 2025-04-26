package redis

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"testing"
	"time"

	mock_redis "Crypto.com/mocks"
	"github.com/golang/mock/gomock"
	"github.com/redis/go-redis/v9"
)

func TestCacheRepository(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_redis.NewMockCmdable(ctrl)
	logger := log.New(&testLogWriter{t}, "TEST: ", log.LstdFlags)
	repo := NewCacheRepository(mockClient, 30*time.Minute, logger)

	t.Run("GetBalance cache miss", func(t *testing.T) {
		mockClient.EXPECT().Get(gomock.Any(), "balance:user1").Return(redis.NewStringResult("", redis.Nil))

		balance, err := repo.GetBalance(context.Background(), "user1")
		if !errors.Is(err, redis.Nil) {
			t.Errorf("Expected redis.Nil error, got %v", err)
		}
		if balance != 0 {
			t.Errorf("Expected 0 balance, got %f", balance)
		}
	})

	t.Run("GetBalance redis error", func(t *testing.T) {
		mockErr := errors.New("connection failed")
		mockClient.EXPECT().Get(gomock.Any(), "balance:user1").Return(redis.NewStringResult("", mockErr))

		_, err := repo.GetBalance(context.Background(), "user1")
		if !errors.Is(err, mockErr) {
			t.Errorf("Expected connection error, got %v", err)
		}
	})

	t.Run("GetBalance valid value", func(t *testing.T) {
		expected := 99.99
		serialized, _ := json.Marshal(expected)
		mockClient.EXPECT().Get(gomock.Any(), "balance:user1").Return(redis.NewStringResult(string(serialized), nil))

		balance, err := repo.GetBalance(context.Background(), "user1")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if balance != expected {
			t.Errorf("Expected %f, got %f", expected, balance)
		}
	})

	t.Run("SetBalance success", func(t *testing.T) {
		val, _ := json.Marshal(50.0)
		mockClient.EXPECT().Set(gomock.Any(), "balance:user2", val, 30*time.Minute).Return(redis.NewStatusResult("OK", nil))

		err := repo.SetBalance(context.Background(), "user2", 50.0)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("InvalidateBalance success", func(t *testing.T) {
		mockClient.EXPECT().Del(gomock.Any(), "balance:user3").Return(redis.NewIntResult(1, nil))

		err := repo.InvalidateBalance(context.Background(), "user3")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

type testLogWriter struct {
	t *testing.T
}

func (w *testLogWriter) Write(p []byte) (n int, err error) {
	w.t.Helper()
	w.t.Log(string(p))
	return len(p), nil
}
