package domain

import (
	"context"
	"time"
)

type Device struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(100)"`
	DeviceID     string    `json:"device_id" gorm:"type:varchar(100);uniqueIndex;not null"`
	DeviceName   string    `json:"device_name" gorm:"type:varchar(255)"`
	DeviceStatus string    `json:"device_status" gorm:"type:varchar(50)"`
	IPAddress    string    `json:"ip_address" gorm:"type:varchar(50)"`
	Port         string    `json:"port" gorm:"type:varchar(10)"`
	UserEmail    string    `json:"user_email" gorm:"type:varchar(255);index"`
	LastSeen     time.Time `json:"last_seen"`
}

type DeviceRepository interface {
	GetAllDevices(ctx context.Context) ([]*Device, error)
	GetByID(ctx context.Context, id string) (*Device, error)
	Update(ctx context.Context, id string, device *Device) error
	Save(ctx context.Context, device *Device) error
}
