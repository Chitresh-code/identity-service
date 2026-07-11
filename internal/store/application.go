package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/sales-intelligence1/identity-service/internal/application"
)

type applicationModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (applicationModel) TableName() string { return "applications" }

func (m applicationModel) toDomain() application.Application {
	return application.Application{ID: m.ID, Name: m.Name, CreatedAt: m.CreatedAt}
}

// ApplicationStore persists applications in Postgres via GORM.
type ApplicationStore struct {
	db *gorm.DB
}

// NewApplicationStore builds an ApplicationStore backed by db.
func NewApplicationStore(db *gorm.DB) *ApplicationStore {
	return &ApplicationStore{db: db}
}

// Create implements application.Store.
func (s *ApplicationStore) Create(ctx context.Context, name string) (application.Application, error) {
	id, err := newID()
	if err != nil {
		return application.Application{}, err
	}

	m := applicationModel{ID: id, Name: name}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return application.Application{}, fmt.Errorf("create application %q: %w", name, err)
	}
	return m.toDomain(), nil
}

// List implements application.Store.
func (s *ApplicationStore) List(ctx context.Context) ([]application.Application, error) {
	var ms []applicationModel
	if err := s.db.WithContext(ctx).Order("created_at desc").Find(&ms).Error; err != nil {
		return nil, fmt.Errorf("list applications: %w", err)
	}
	apps := make([]application.Application, len(ms))
	for i, m := range ms {
		apps[i] = m.toDomain()
	}
	return apps, nil
}

// ByID implements application.Store.
func (s *ApplicationStore) ByID(ctx context.Context, id string) (application.Application, error) {
	var m applicationModel
	if err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return application.Application{}, fmt.Errorf("load application %q: %w", id, err)
	}
	return m.toDomain(), nil
}
