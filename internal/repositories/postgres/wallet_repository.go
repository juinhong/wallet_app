package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/sirupsen/logrus"
	"time"

	"Crypto.com/internal/models"
)

type WalletRepository interface {
	Deposit(ctx context.Context, userID string, amount float64) error
	Withdraw(ctx context.Context, userID string, amount float64) error
	Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error
	GetBalance(ctx context.Context, userID string) (float64, error)
	GetTransactionHistory(ctx context.Context, userID string, limit, offset int) ([]models.Transaction, error)
}

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidAmount       = errors.New("invalid amount")
	ErrInvalidUserID       = errors.New("invalid user ID")
	ErrInsufficientFunds   = errors.New("insufficient funds")
)

// Implement the interface
type PostgresWalletRepository struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewWalletRepository(db *sql.DB, logger *logrus.Logger) *PostgresWalletRepository {
	return &PostgresWalletRepository{db: db, logger: logger}
}

// Deposit adds amount to user's balance and creates transaction record
func (r *PostgresWalletRepository) Deposit(ctx context.Context, userID string, amount float64) error {
	if userID == "" {
		return ErrInvalidUserID
	}

	if amount <= 0 {
		return ErrInvalidAmount
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
		"amount": amount,
	})

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Deposit transaction failed")
		return err
	}
	defer tx.Rollback()

	// Update balance - create wallet if not exists
	_, err = tx.ExecContext(ctx,
		`INSERT INTO wallets (user_id, balance) 
        VALUES ($1, $2)
        ON CONFLICT (user_id) 
        DO UPDATE SET balance = wallets.balance + $2`,
		userID, amount,
	)
	if err != nil {
		return err
	}

	// Create transaction record
	_, err = tx.ExecContext(ctx,
		`INSERT INTO transactions 
		(from_user_id, amount, type, created_at) 
		VALUES ($1, $2, $3, $4)`,
		userID, amount, "deposit", time.Now(),
	)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	logger.Info("Deposit successful")
	return nil
}

// Withdraw deducts amount from user's balance if sufficient funds
func (r *PostgresWalletRepository) Withdraw(ctx context.Context, userID string, amount float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	err = tx.QueryRowContext(ctx,
		"SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE",
		userID,
	).Scan(&currentBalance)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	if currentBalance < amount {
		return ErrInsufficientBalance
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance - $1 WHERE user_id = $2",
		amount, userID,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO transactions 
		(from_user_id, amount, type, created_at) 
		VALUES ($1, $2, $3, $4)`,
		userID, amount, "withdrawal", time.Now(),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Transfer moves funds between two users atomically
func (r *PostgresWalletRepository) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check and deduct from sender
	var currentBalance float64
	err = tx.QueryRowContext(ctx,
		"SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE",
		fromUserID,
	).Scan(&currentBalance)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrUserNotFound
	}
	if err != nil {
		return err
	}

	if currentBalance < amount {
		return ErrInsufficientBalance
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance - $1 WHERE user_id = $2",
		amount, fromUserID,
	)
	if err != nil {
		return err
	}

	// Add to receiver
	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance + $1 WHERE user_id = $2",
		amount, toUserID,
	)
	if err != nil {
		return err
	}

	// Create transaction records
	now := time.Now()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO transactions 
		(from_user_id, to_user_id, amount, type, created_at) 
		VALUES ($1, $2, $3, $4, $5)`,
		fromUserID, toUserID, amount, "transfer", now,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetBalance returns current wallet balance
func (r *PostgresWalletRepository) GetBalance(ctx context.Context, userID string) (float64, error) {
	var balance float64
	err := r.db.QueryRowContext(ctx,
		"SELECT balance FROM wallets WHERE user_id = $1",
		userID,
	).Scan(&balance)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrUserNotFound
	}
	return balance, err
}

// GetTransactionHistory returns paginated transaction history
func (r *PostgresWalletRepository) GetTransactionHistory(ctx context.Context, userID string, limit, offset int) ([]models.Transaction, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, from_user_id, to_user_id, amount, type, created_at 
		FROM transactions 
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var txn models.Transaction
		err := rows.Scan(
			&txn.ID,
			&txn.FromUserID,
			&txn.ToUserID,
			&txn.Amount,
			&txn.Type,
			&txn.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, txn)
	}
	return transactions, nil
}
