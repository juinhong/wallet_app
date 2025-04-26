package models

import "time"

type Transaction struct {
	ID         *string    `json:"id,omitempty"`
	FromUserID *string    `json:"from_user_id,omitempty"`
	ToUserID   *string    `json:"to_user_id,omitempty"`
	Amount     *float64   `json:"amount,omitempty"`
	Type       *string    `json:"type,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
}
