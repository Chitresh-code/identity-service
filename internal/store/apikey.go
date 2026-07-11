package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/internal/apikey"
)

type apiKeyModel struct {
	ID            string    `gorm:"column:id;primaryKey"`
	ApplicationID string    `gorm:"column:application_id"`
	Prefix        string    `gorm:"column:prefix;uniqueIndex"`
	SecretHash    string    `gorm:"column:secret_hash"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (apiKeyModel) TableName() string { return "api_keys" }

func (m apiKeyModel) toDomain() apikey.APIKey {
	return apikey.APIKey{
		ID:            m.ID,
		ApplicationID: m.ApplicationID,
		Prefix:        m.Prefix,
		CreatedAt:     m.CreatedAt,
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
