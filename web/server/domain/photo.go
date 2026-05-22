package domain

import (
	"context"
	"time"
)

type Photo struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(100)"`
	Timestamp    time.Time `json:"timestamp" gorm:"index"`
	ImageType    string    `json:"image_type" gorm:"type:varchar(50)"`
	PresignedURL string    `json:"presigned_url" gorm:"-"` // Not stored in DB, generated on the fly
	DeviceID     string    `json:"device_id" gorm:"type:varchar(100);index"`
	Text         string    `json:"text" gorm:"type:text"`

	// Medical Data Fields - Stored as a single JSONB column
	MedicalData MedicalData `json:"medical_data" gorm:"column:medical_data;type:jsonb;serializer:json"`
}

type PhotoRepository interface {
	GetPhotos(ctx context.Context, filters map[string]any) ([]*Photo, error)
	GetByID(ctx context.Context, id string) (*Photo, error)
	Save(ctx context.Context, photo *Photo) error
	Update(ctx context.Context, id string, update any) error
	Delete(ctx context.Context, id string) error
	DeleteAll(ctx context.Context) (int64, error)
}
