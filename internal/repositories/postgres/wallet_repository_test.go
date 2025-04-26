package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWalletRepository(t *testing.T) {
	ctx := context.Background()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	logger := logrus.New()
	repo := NewWalletRepository(mockDB, logger)

	t.Run("Deposit", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectExec(`INSERT INTO wallets`).WithArgs("user1", 100.0).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectExec(`INSERT INTO transactions`).WithArgs("user1", 100.0, "deposit", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			require.NoError(t, repo.Deposit(ctx, "user1", 100.0))
		})

		t.Run("invalid amount", func(t *testing.T) {
			err := repo.Deposit(ctx, "user1", -50.0)
			require.ErrorIs(t, err, ErrInvalidAmount)
		})
	})

	t.Run("Withdraw", func(t *testing.T) {
		t.Run("insufficient balance", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT balance`).WithArgs("user1").WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(50.0))
			mock.ExpectRollback()
			err := repo.Withdraw(ctx, "user1", 100.0)
			require.ErrorIs(t, err, ErrInsufficientBalance)
		})

		t.Run("user not found", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT balance`).WithArgs("invalid").WillReturnError(sql.ErrNoRows)
			mock.ExpectRollback()
			err := repo.Withdraw(ctx, "invalid", 100.0)
			require.ErrorIs(t, err, ErrUserNotFound)
		})
	})

	t.Run("Transfer", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			mock.ExpectBegin()
			mock.ExpectQuery(`SELECT balance`).WithArgs("user1").WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(200.0))
			mock.ExpectExec(`UPDATE wallets`).WithArgs(100.0, "user1").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`UPDATE wallets`).WithArgs(100.0, "user2").WillReturnResult(sqlmock.NewResult(0, 1))
			mock.ExpectExec(`INSERT INTO transactions`).WithArgs("user1", "user2", 100.0, "transfer", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(1, 1))
			mock.ExpectCommit()
			require.NoError(t, repo.Transfer(ctx, "user1", "user2", 100.0))
		})
	})

	t.Run("GetBalance", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			mock.ExpectQuery(`SELECT balance`).WithArgs("user1").WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(150.0))
			balance, err := repo.GetBalance(ctx, "user1")
			require.NoError(t, err)
			require.Equal(t, 150.0, balance)
		})

		t.Run("user not found", func(t *testing.T) {
			mock.ExpectQuery(`SELECT balance`).WithArgs("invalid").WillReturnError(sql.ErrNoRows)
			_, err := repo.GetBalance(ctx, "invalid")
			require.ErrorIs(t, err, ErrUserNotFound)
		})
	})

	t.Run("GetTransactionHistory", func(t *testing.T) {
		now := time.Now()
		t.Run("success", func(t *testing.T) {
			mock.ExpectQuery(`SELECT`).WithArgs("user1", 10, 0).WillReturnRows(sqlmock.NewRows(
				[]string{"id", "from_user_id", "to_user_id", "amount", "type", "created_at"},
			).AddRow(1, "user1", "", 100.0, "deposit", now).AddRow(2, "user1", "user2", 50.0, "transfer", now))

			txns, err := repo.GetTransactionHistory(ctx, "user1", 10, 0)
			require.NoError(t, err)
			require.Len(t, txns, 2)
			require.Equal(t, "deposit", *txns[0].Type)
		})

		t.Run("query error", func(t *testing.T) {
			mock.ExpectQuery(`SELECT`).WithArgs("user1", 10, 0).WillReturnError(errors.New("query error"))
			_, err := repo.GetTransactionHistory(ctx, "user1", 10, 0)
			require.ErrorContains(t, err, "query error")
		})
	})
}
