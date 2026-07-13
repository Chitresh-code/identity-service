package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/pkg/handoff"
)

type handoffModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	UserID    string    `gorm:"column:user_id"`
	CodeHash  string    `gorm:"column:code_hash;uniqueIndex"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (handoffModel) TableName() string { return "login_handoff_codes" }

func (m handoffModel) toDomain() handoff.Handoff {
	return handoff.Handoff{ID: m.ID, UserID: m.UserID, ExpiresAt: m.ExpiresAt}
}

// HandoffStore persists login handoff codes in Postgres via GORM.
type HandoffStore struct {
	db *gorm.DB
}

// NewHandoffStore builds a HandoffStore backed by db.
func NewHandoffStore(db *gorm.DB) *HandoffStore {
	return &HandoffStore{db: db}
}

// Create implements handoff.Store.
func (s *HandoffStore) Create(ctx context.Context, userID, codeHash string, expiresAt time.Time) (handoff.Handoff, error) {
	id, err := newID()
	if err != nil {
		return handoff.Handoff{}, err
	}

	m := handoffModel{ID: id, UserID: userID, CodeHash: codeHash, ExpiresAt: expiresAt}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return handoff.Handoff{}, fmt.Errorf("create handoff code for user %q: %w", userID, err)
	}
	return m.toDomain(), nil
}

// Consume implements handoff.Store. A single DELETE...RETURNING statement
// makes lookup and single-use spend atomic -- two concurrent exchanges of
// the same code can't both succeed.
func (s *HandoffStore) Consume(ctx context.Context, codeHash string) (handoff.Handoff, error) {
	var m handoffModel
	err := s.db.WithContext(ctx).Raw(
		`DELETE FROM login_handoff_codes WHERE code_hash = ? AND expires_at > ? RETURNING id, user_id, expires_at, created_at`,
		codeHash, time.Now(),
	).Scan(&m).Error
	if err != nil {
		return handoff.Handoff{}, fmt.Errorf("consume handoff code: %w", err)
	}
	if m.ID == "" {
		return handoff.Handoff{}, handoff.ErrNotFound
	}
	return m.toDomain(), nil
}
