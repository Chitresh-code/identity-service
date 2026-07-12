package store

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/pkg/signingkey"
)

type signingKeyModel struct {
	ID            string    `gorm:"column:id;primaryKey"`
	PrivateKeyPEM string    `gorm:"column:private_key_pem"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (signingKeyModel) TableName() string { return "signing_keys" }

func (m signingKeyModel) toDomain() (signingkey.Key, error) {
	block, _ := pem.Decode([]byte(m.PrivateKeyPEM))
	if block == nil {
		return signingkey.Key{}, fmt.Errorf("decode pem for signing key %q", m.ID)
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return signingkey.Key{}, fmt.Errorf("parse signing key %q: %w", m.ID, err)
	}
	return signingkey.Key{ID: m.ID, PrivateKey: priv, CreatedAt: m.CreatedAt}, nil
}

func signingKeyModelFromDomain(k signingkey.Key) signingKeyModel {
	der := x509.MarshalPKCS1PrivateKey(k.PrivateKey)
	block := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	return signingKeyModel{ID: k.ID, PrivateKeyPEM: string(block), CreatedAt: k.CreatedAt}
}

// SigningKeyStore persists JWT signing keys in Postgres via GORM.
type SigningKeyStore struct {
	db *gorm.DB
}

// NewSigningKeyStore builds a SigningKeyStore backed by db.
func NewSigningKeyStore(db *gorm.DB) *SigningKeyStore {
	return &SigningKeyStore{db: db}
}

// Latest implements signingkey.Store.
func (s *SigningKeyStore) Latest(ctx context.Context) (signingkey.Key, error) {
	var m signingKeyModel
	err := s.db.WithContext(ctx).Order("created_at desc").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return signingkey.Key{}, signingkey.ErrNoKeys
		}
		return signingkey.Key{}, fmt.Errorf("load latest signing key: %w", err)
	}
	return m.toDomain()
}

// All implements signingkey.Store.
func (s *SigningKeyStore) All(ctx context.Context) ([]signingkey.Key, error) {
	var ms []signingKeyModel
	if err := s.db.WithContext(ctx).Order("created_at desc").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("list signing keys: %w", err)
	}

	keys := make([]signingkey.Key, 0, len(ms))
	for _, m := range ms {
		k, err := m.toDomain()
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// Save implements signingkey.Store.
func (s *SigningKeyStore) Save(ctx context.Context, k signingkey.Key) error {
	m := signingKeyModelFromDomain(k)
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("save signing key %q: %w", k.ID, err)
	}
	return nil
}
