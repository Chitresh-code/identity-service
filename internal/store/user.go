package store

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/sales-intelligence1/identity-service/internal/user"
)

type userModel struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Auth0Sub  string    `gorm:"column:auth0_sub;uniqueIndex"`
	Email     string    `gorm:"column:email"`
	Name      string    `gorm:"column:name"`
	IsAdmin   bool      `gorm:"column:is_admin"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (userModel) TableName() string { return "users" }

func (m userModel) toDomain() user.User {
	return user.User{
		ID:        m.ID,
		Auth0Sub:  m.Auth0Sub,
		Email:     m.Email,
		Name:      m.Name,
		IsAdmin:   m.IsAdmin,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// UserStore persists users in Postgres via GORM.
type UserStore struct {
	db *gorm.DB
}

// NewUserStore builds a UserStore backed by db.
func NewUserStore(db *gorm.DB) *UserStore {
	return &UserStore{db: db}
}

// UpsertByAuth0Sub implements user.Store.
func (s *UserStore) UpsertByAuth0Sub(ctx context.Context, sub, email, name string) (user.User, error) {
	id, err := newID()
	if err != nil {
		return user.User{}, err
	}

	// Only the very first user ever created becomes admin. This is set on the
	// model but deliberately left out of DoUpdates below, so an existing user's
	// admin status is never touched by a later login.
	var userCount int64
	if err := s.db.WithContext(ctx).Model(&userModel{}).Count(&userCount).Error; err != nil {
		return user.User{}, fmt.Errorf("count users: %w", err)
	}

	m := userModel{ID: id, Auth0Sub: sub, Email: email, Name: name, IsAdmin: userCount == 0}
	err = s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "auth0_sub"}},
			DoUpdates: clause.AssignmentColumns([]string{"email", "name", "updated_at"}),
		}).
		Create(&m).Error
	if err != nil {
		return user.User{}, fmt.Errorf("upsert user %q: %w", sub, err)
	}

	// The row above may be an existing user (conflict path), in which case m's
	// generated id isn't the one actually stored -- read the canonical row back
	// rather than relying on GORM's OnConflict+RETURNING behavior.
	var stored userModel
	if err := s.db.WithContext(ctx).First(&stored, "auth0_sub = ?", sub).Error; err != nil {
		return user.User{}, fmt.Errorf("load user %q after upsert: %w", sub, err)
	}
	return stored.toDomain(), nil
}

// ByID implements user.Store.
func (s *UserStore) ByID(ctx context.Context, id string) (user.User, error) {
	var m userModel
	if err := s.db.WithContext(ctx).First(&m, "id = ?", id).Error; err != nil {
		return user.User{}, fmt.Errorf("load user %q: %w", id, err)
	}
	return m.toDomain(), nil
}
