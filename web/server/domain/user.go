package domain

import "context"

type User struct {
	Email    string `json:"email" gorm:"primaryKey;type:varchar(255)"`
	Password string `json:"password,omitempty" gorm:"type:varchar(255);not null"`
	Role     string `json:"role,omitempty" gorm:"type:varchar(50);default:'user'"`
}

type UserRepository interface {
	Save(ctx context.Context, email, password, role string) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
}
