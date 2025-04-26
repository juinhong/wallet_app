package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sirupsen/logrus"

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
	ErrInvalidLimit        = errors.New("invalid limit")
)

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
		r.logger.Warn("Deposit - userID cannot be an empty string")
		return ErrInvalidUserID
	}

	if amount <= 0 {
		r.logger.Warn("Deposit - amount cannot be less than zero")
		return ErrInvalidAmount
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
		"amount": amount,
	})

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Deposit - Begin DB transaction failed")
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
		logger.WithError(err).Error("Deposit - Update balance failed")
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
		logger.WithError(err).Error("Deposit - Create transaction record failed")
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.WithError(err).Error("Deposit - Commit DB transaction failed")
		return err
	}

	logger.Info("Deposit successful")
	return nil
}

// Withdraw deducts amount from user's balance if sufficient funds
func (r *PostgresWalletRepository) Withdraw(ctx context.Context, userID string, amount float64) error {
	if userID == "" {
		r.logger.Warn("Withdraw - userID cannot be an empty string")
		return ErrInvalidUserID
	}

	if amount <= 0 {
		r.logger.Warn("Withdraw - amount cannot be less than zero")
		return ErrInvalidAmount
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
		"amount": amount,
	})

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		logger.WithError(err).Error("Withdraw - Begin DB transaction failed")
		return err
	}
	defer tx.Rollback()

	var currentBalance float64
	err = tx.QueryRowContext(ctx,
		"SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE",
		userID,
	).Scan(&currentBalance)

	if errors.Is(err, sql.ErrNoRows) {
		logger.WithError(err).Error("Withdraw - Cannot find user in the database")
		return ErrUserNotFound
	}
	if err != nil {
		logger.WithError(err).Error("Withdraw - Query user balance failed")
		return err
	}

	if currentBalance < amount {
		logger.WithError(err).Error("Withdraw - User balance is too low")
		return ErrInsufficientBalance
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance - $1 WHERE user_id = $2",
		amount, userID,
	)
	if err != nil {
		logger.WithError(err).Error("Withdraw - Update user balance failed")
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO transactions 
		(from_user_id, amount, type, created_at) 
		VALUES ($1, $2, $3, $4)`,
		userID, amount, "withdrawal", time.Now(),
	)
	if err != nil {
		logger.WithError(err).Error("Withdraw - Create transaction record failed")
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.WithError(err).Error("Withdraw - Commit DB transaction failed")
		return err
	}

	logger.Info("Withdraw successful")
	return nil
}

// Transfer moves funds between two users atomically
func (r *PostgresWalletRepository) Transfer(ctx context.Context, fromUserID, toUserID string, amount float64) error {
	if fromUserID == "" || toUserID == "" {
		r.logger.Warn("Transfer - fromUserID and toUserID cannot be an empty string")
		return ErrInvalidUserID
	}

	if fromUserID == toUserID {
		r.logger.Warn("Transfer - fromUserID and toUserID cannot be the same")
		return ErrInvalidUserID
	}

	if amount <= 0 {
		r.logger.Warn("Transfer - amount cannot be less than zero")
		return ErrInvalidAmount
	}

	logger := r.logger.WithFields(logrus.Fields{
		"fromUserID": fromUserID,
		"toUserID":   toUserID,
		"amount":     amount,
	})

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		r.logger.WithError(err).Error("Transfer - Begin DB transaction failed")
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
		r.logger.WithError(err).Error("Transfer - Cannot find sender in the database")
		return ErrUserNotFound
	}
	if err != nil {
		logger.WithError(err).Error("Transfer - Query sender balance failed")
		return err
	}

	if currentBalance < amount {
		logger.WithError(err).Error("Transfer - Sender balance is too low")
		return ErrInsufficientBalance
	}

	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance - $1 WHERE user_id = $2",
		amount, fromUserID,
	)
	if err != nil {
		logger.WithError(err).Error("Transfer - Update sender balance failed")
		return err
	}

	// Add to receiver
	_, err = tx.ExecContext(ctx,
		"UPDATE wallets SET balance = balance + $1 WHERE user_id = $2",
		amount, toUserID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		r.logger.WithError(err).Error("Transfer - Cannot find receiver in the database")
		return ErrUserNotFound
	}

	if err != nil {
		logger.WithError(err).Error("Transfer - Update receiver balance failed")
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
		logger.WithError(err).Error("Transfer - Create transaction record failed")
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.WithError(err).Error("Transfer - Commit DB transaction failed")
		return err
	}

	logger.Info("Transfer successful")
	return nil
}

// GetBalance returns current wallet balance
func (r *PostgresWalletRepository) GetBalance(ctx context.Context, userID string) (float64, error) {
	if userID == "" {
		r.logger.Warn("GetBalance - userID cannot be an empty string")
		return 0, ErrInvalidUserID
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
	})

	var balance float64
	err := r.db.QueryRowContext(ctx,
		"SELECT balance FROM wallets WHERE user_id = $1",
		userID,
	).Scan(&balance)

	if errors.Is(err, sql.ErrNoRows) {
		logger.WithError(err).Error("GetBalance - Cannot user in database")
		return 0, ErrUserNotFound
	}

	if err != nil {
		logger.WithError(err).Error("GetBalance - Query user balance failed")
		return 0, err
	}

	return balance, nil
}

// GetTransactionHistory returns paginated transaction history
func (r *PostgresWalletRepository) GetTransactionHistory(ctx context.Context, userID string, limit, offset int) ([]models.Transaction, error) {
	if userID == "" {
		r.logger.Warn("GetTransactionHistory - userID cannot be an empty string")
		return nil, ErrInvalidUserID
	}

	if limit <= 0 {
		r.logger.Warn("GetTransactionHistory - limit cannot be less than 0")
		return nil, ErrInvalidLimit
	}

	logger := r.logger.WithFields(logrus.Fields{
		"userID": userID,
	})

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, from_user_id, to_user_id, amount, type, created_at 
		FROM transactions 
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		logger.WithError(err).Error("GetTransactionHistory - Query transactions failed")
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
			logger.WithError(err).Error("GetTransactionHistory - Scan transactions failed")
			return nil, err
		}
		transactions = append(transactions, txn)
	}
	return transactions, nil
}
