package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/pkg/session"
)

type sessionModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	UserID    string    `gorm:"column:user_id"`
	TokenHash string    `gorm:"column:token_hash;uniqueIndex"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (sessionModel) TableName() string { return "sessions" }

func (m sessionModel) toDomain() session.Session {
	return session.Session{
		ID:        m.ID,
		UserID:    m.UserID,
		ExpiresAt: m.ExpiresAt,
		CreatedAt: m.CreatedAt,
	}
}

// SessionStore persists sessions in Postgres via GORM.
type SessionStore struct {
	db *gorm.DB
}

// NewSessionStore builds a SessionStore backed by db.
func NewSessionStore(db *gorm.DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create implements session.Store.
func (s *SessionStore) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (session.Session, error) {
	id, err := newID()
	if err != nil {
		return session.Session{}, err
	}

	m := sessionModel{ID: id, UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return session.Session{}, fmt.Errorf("create session for user %q: %w", userID, err)
	}
	return m.toDomain(), nil
}

// FindByTokenHash implements session.Store. It only returns sessions that
// haven't expired yet.
func (s *SessionStore) FindByTokenHash(ctx context.Context, tokenHash string) (session.Session, error) {
	var m sessionModel
	err := s.db.WithContext(ctx).
		First(&m, "token_hash = ? AND expires_at > ?", tokenHash, time.Now()).Error
	if err != nil {
		return session.Session{}, fmt.Errorf("find session: %w", err)
	}
	return m.toDomain(), nil
}

// DeleteByTokenHash implements session.Store.
func (s *SessionStore) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	if err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&sessionModel{}).Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
