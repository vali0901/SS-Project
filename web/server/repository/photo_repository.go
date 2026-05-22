package repository

import (
	"context"
	"mqtt-streaming-server/domain"
	"time"

	"gorm.io/gorm"
)

type photoRepository struct {
	db *gorm.DB
}

func NewPhotoRepository(db *gorm.DB) *photoRepository {
	return &photoRepository{db: db}
}

func (r *photoRepository) GetPhotos(ctx context.Context, filters map[string]any) ([]*domain.Photo, error) {
	query := r.db.WithContext(ctx).Model(&domain.Photo{})

	if userEmail, ok := filters["user_email"].(string); ok && userEmail != "" {
		query = query.Where("user_email = ?", userEmail)
	}

	if deviceID, ok := filters["device_id"].(string); ok && deviceID != "" {
		query = query.Where("device_id = ?", deviceID)
	}

	if startDate, ok := filters["start_date"].(time.Time); ok && !startDate.IsZero() {
		query = query.Where("timestamp >= ?", startDate)
	}

	if endDate, ok := filters["end_date"].(time.Time); ok && !endDate.IsZero() {
		query = query.Where("timestamp <= ?", endDate)
	}

	if searchText, ok := filters["text"].(string); ok && searchText != "" {
		// PostgreSQL full-text search or simple LIKE for simplicity here
		query = query.Where("text ILIKE ?", "%"+searchText+"%")
	}

	var photos []*domain.Photo
	err := query.Order("timestamp DESC").Find(&photos).Error
	return photos, err
}

func (r *photoRepository) GetByID(ctx context.Context, id string) (*domain.Photo, error) {
	var photo domain.Photo
	result := r.db.WithContext(ctx).Where("id = ?", id).Limit(1).Find(&photo)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &photo, nil
}

func (r *photoRepository) Save(ctx context.Context, photo *domain.Photo) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

func (r *photoRepository) Update(ctx context.Context, id string, update any) error {
	// If the update is a map, we need to handle medical_data specially if it's not present
	// or if we want to merge it. For now, we expect the handler to provide the full
	// updated object or a map that GORM can handle.
	return r.db.WithContext(ctx).Model(&domain.Photo{}).Where("id = ?", id).Updates(update).Error
}

func (r *photoRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Photo{}, "id = ?", id).Error
}

func (r *photoRepository) DeleteAll(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&domain.Photo{})
	return result.RowsAffected, result.Error
}
