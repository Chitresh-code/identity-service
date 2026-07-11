package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/internal/apikey"
)

type apiKeyModel struct {
	ID            string     `gorm:"column:id;primaryKey"`
	ApplicationID string     `gorm:"column:application_id"`
	Prefix        string     `gorm:"column:prefix;uniqueIndex"`
	SecretHash    string     `gorm:"column:secret_hash"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime"`
	RevokedAt     *time.Time `gorm:"column:revoked_at"`
}

func (apiKeyModel) TableName() string { return "api_keys" }

func (m apiKeyModel) toDomain() apikey.APIKey {
	return apikey.APIKey{
		ID:            m.ID,
		ApplicationID: m.ApplicationID,
		Prefix:        m.Prefix,
		CreatedAt:     m.CreatedAt,
		RevokedAt:     m.RevokedAt,
	}
}

// APIKeyStore persists API keys in Postgres via GORM.
type APIKeyStore struct {
	db *gorm.DB
}

// NewAPIKeyStore builds an APIKeyStore backed by db.
func NewAPIKeyStore(db *gorm.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

// Create implements apikey.Store.
func (s *APIKeyStore) Create(ctx context.Context, applicationID, prefix, secretHash string) (apikey.APIKey, error) {
	id, err := newID()
	if err != nil {
		return apikey.APIKey{}, err
	}

	m := apiKeyModel{ID: id, ApplicationID: applicationID, Prefix: prefix, SecretHash: secretHash}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return apikey.APIKey{}, fmt.Errorf("create api key for application %q: %w", applicationID, err)
	}
	return m.toDomain(), nil
}

// List implements apikey.Store.
func (s *APIKeyStore) List(ctx context.Context, applicationID string) ([]apikey.APIKey, error) {
	var ms []apiKeyModel
	err := s.db.WithContext(ctx).
		Where("application_id = ?", applicationID).
		Order("created_at desc").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("list api keys for application %q: %w", applicationID, err)
	}
	keys := make([]apikey.APIKey, len(ms))
	for i, m := range ms {
		keys[i] = m.toDomain()
	}
	return keys, nil
}

// Rotate implements apikey.Store: atomically issues a new key for
// applicationID and revokes oldKeyID, so a partial failure never leaves an
// application with zero live keys or two live keys unexpectedly.
func (s *APIKeyStore) Rotate(ctx context.Context, oldKeyID, applicationID, newPrefix, newSecretHash string) (apikey.APIKey, error) {
	id, err := newID()
	if err != nil {
		return apikey.APIKey{}, err
	}

	var newKey apiKeyModel
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var old apiKeyModel
		if err := tx.First(&old, "id = ? AND application_id = ?", oldKeyID, applicationID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apikey.ErrKeyNotFound
			}
			return fmt.Errorf("load api key %q: %w", oldKeyID, err)
		}
		if old.RevokedAt != nil {
			return apikey.ErrKeyRevoked
		}

		newKey = apiKeyModel{ID: id, ApplicationID: applicationID, Prefix: newPrefix, SecretHash: newSecretHash}
		if err := tx.Create(&newKey).Error; err != nil {
			return fmt.Errorf("create rotated api key: %w", err)
		}

		now := time.Now()
		if err := tx.Model(&old).Update("revoked_at", now).Error; err != nil {
			return fmt.Errorf("revoke api key %q: %w", oldKeyID, err)
		}
		return nil
	})
	if err != nil {
		return apikey.APIKey{}, err
	}
	return newKey.toDomain(), nil
}

// Revoke implements apikey.Store.
func (s *APIKeyStore) Revoke(ctx context.Context, id, applicationID string) error {
	var m apiKeyModel
	if err := s.db.WithContext(ctx).First(&m, "id = ? AND application_id = ?", id, applicationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apikey.ErrKeyNotFound
		}
		return fmt.Errorf("load api key %q: %w", id, err)
	}
	if m.RevokedAt != nil {
		return nil
	}

	if err := s.db.WithContext(ctx).Model(&m).Update("revoked_at", time.Now()).Error; err != nil {
		return fmt.Errorf("revoke api key %q: %w", id, err)
	}
	return nil
}
