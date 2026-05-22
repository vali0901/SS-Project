package repository

import (
	"context"
	"mqtt-streaming-server/domain"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type deviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *deviceRepository {
	return &deviceRepository{db: db}
}

func (r *deviceRepository) GetAllDevices(ctx context.Context) ([]*domain.Device, error) {
	var devices []*domain.Device
	err := r.db.WithContext(ctx).Find(&devices).Error
	return devices, err
}

func (r *deviceRepository) GetByID(ctx context.Context, id string) (*domain.Device, error) {
	var device domain.Device
	result := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&device)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &device, nil
}

func (r *deviceRepository) Update(ctx context.Context, id string, device *domain.Device) error {
	return r.db.WithContext(ctx).Model(&domain.Device{}).Where("id = ?", id).Updates(device).Error
}

func (r *deviceRepository) Save(ctx context.Context, device *domain.Device) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(device).Error
}
