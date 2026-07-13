package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/pkg/userrefresh"
)

type userRefreshTokenModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	UserID    string    `gorm:"column:user_id"`
	TokenHash string    `gorm:"column:token_hash;uniqueIndex"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (userRefreshTokenModel) TableName() string { return "user_refresh_tokens" }

func (m userRefreshTokenModel) toDomain() userrefresh.Token {
	return userrefresh.Token{ID: m.ID, UserID: m.UserID, ExpiresAt: m.ExpiresAt}
}

// UserRefreshTokenStore persists user refresh tokens in Postgres via GORM.
type UserRefreshTokenStore struct {
	db *gorm.DB
}

// NewUserRefreshTokenStore builds a UserRefreshTokenStore backed by db.
func NewUserRefreshTokenStore(db *gorm.DB) *UserRefreshTokenStore {
	return &UserRefreshTokenStore{db: db}
}

// Create implements userrefresh.Store.
func (s *UserRefreshTokenStore) Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (userrefresh.Token, error) {
	id, err := newID()
	if err != nil {
		return userrefresh.Token{}, err
	}

	m := userRefreshTokenModel{ID: id, UserID: userID, TokenHash: tokenHash, ExpiresAt: expiresAt}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return userrefresh.Token{}, fmt.Errorf("create refresh token for user %q: %w", userID, err)
	}
	return m.toDomain(), nil
}

// FindByTokenHash implements userrefresh.Store.
func (s *UserRefreshTokenStore) FindByTokenHash(ctx context.Context, tokenHash string) (userrefresh.Token, error) {
	var m userRefreshTokenModel
	err := s.db.WithContext(ctx).
		First(&m, "token_hash = ? AND expires_at > ?", tokenHash, time.Now()).Error
	if err != nil {
		return userrefresh.Token{}, fmt.Errorf("find refresh token: %w", err)
	}
	return m.toDomain(), nil
}

// DeleteByTokenHash implements userrefresh.Store.
func (s *UserRefreshTokenStore) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	if err := s.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&userRefreshTokenModel{}).Error; err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
