package repository

import (
	"context"
	"mqtt-streaming-server/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Save(ctx context.Context, email, password, role string) error {
	if role == "" {
		role = "user"
	}
	user := &domain.User{
		Email:    email,
		Password: password,
		Role:     role,
	}
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	result := r.db.WithContext(ctx).Where("email = ?", email).Limit(1).Find(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &user, nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	var users []*domain.User
	err := r.db.WithContext(ctx).Find(&users).Error
	return users, err
}
