package repository

import (
	"context"
	"fmt"
	"mqtt-streaming-server/domain"
	"os"

	"mqtt-streaming-server/crypto"

	"gorm.io/gorm"
)

type photoRepository struct {
	db     *gorm.DB
	crypto *crypto.Service
}

func NewPhotoRepository(db *gorm.DB) *photoRepository {
	encryptionKey, err := os.ReadFile("/run/secrets/encryption.key")
	if err != nil {
		fmt.Printf("Warning: encryption.key not found, using default key: %v\n", err)
		encryptionKey = []byte("default-secret-key-for-development")
	}
	return &photoRepository{db: db, crypto: crypto.NewService(encryptionKey)}
}

func (r *photoRepository) GetPhotos(ctx context.Context, filters map[string]any) ([]*domain.Photo, error) {
	query := r.db.WithContext(ctx).Model(&domain.Photo{})

	if userEmail, ok := filters["user_email"].(string); ok && userEmail != "" {
		query = query.Where("user_email = ?", userEmail)
	}

	var photos []*domain.Photo
	if err := query.Order("timestamp DESC").Find(&photos).Error; err != nil {
		return nil, err
	}

	for _, p := range photos {

		// decrypt text
		if len(p.TextEnc) > 0 {
			text, err := r.crypto.Decrypt(p.TextEnc)
			if err == nil {
				p.Text = string(text)
			}
		}

		// decrypt medical data
		if len(p.MedicalDataEnc) > 0 {
			_ = r.crypto.DecryptStruct(p.MedicalDataEnc, &p.MedicalData)
		}
	}

	return photos, nil
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

	// decrypt text
	if len(photo.TextEnc) > 0 {
		text, err := r.crypto.Decrypt(photo.TextEnc)
		if err == nil {
			photo.Text = string(text)
		}
	}

	// decrypt medical data
	if len(photo.MedicalDataEnc) > 0 {
		err := r.crypto.DecryptStruct(photo.MedicalDataEnc, &photo.MedicalData)
		if err != nil {
			return nil, err
		}
	}

	return &photo, nil
}

func (r *photoRepository) Save(ctx context.Context, photo *domain.Photo) error {

	textEnc, err := r.crypto.Encrypt([]byte(photo.Text))
	if err != nil {
		return err
	}

	medicalEnc, err := r.crypto.EncryptStruct(photo.MedicalData)
	if err != nil {
		return err
	}

	dbPhoto := &domain.Photo{
		ID:        photo.ID,
		Timestamp: photo.Timestamp,
		ImageType: photo.ImageType,
		DeviceID:  photo.DeviceID,
		UserEmail: photo.UserEmail,

		TextEnc:        textEnc,
		MedicalDataEnc: medicalEnc,
	}

	return r.db.WithContext(ctx).Create(dbPhoto).Error
}

func (r *photoRepository) Update(ctx context.Context, id string, update any) error {
	data, ok := update.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid update format: expected map[string]any")
	}

	dbUpdate := map[string]any{}

	// -------------------------
	// TEXT → encrypt into text_enc
	// -------------------------
	if text, ok := data["text"].(string); ok && text != "" {
		encText, err := r.crypto.Encrypt([]byte(text))
		if err != nil {
			return err
		}
		dbUpdate["text_enc"] = encText
	}

	// -------------------------
	// MEDICAL DATA → encrypt into medical_data_enc
	// -------------------------
	if medical, ok := data["medical_data"]; ok && medical != nil {

		encMedical, err := r.crypto.EncryptStruct(medical)
		if err != nil {
			return err
		}

		dbUpdate["medical_data_enc"] = encMedical
	}

	// -------------------------
	// nothing to update
	// -------------------------
	if len(dbUpdate) == 0 {
		return nil
	}

	// -------------------------
	// EXECUTE UPDATE
	// -------------------------
	return r.db.WithContext(ctx).
		Model(&domain.Photo{}).
		Where("id = ?", id).
		Updates(dbUpdate).Error
}

func (r *photoRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&domain.Photo{}, "id = ?", id).Error
}

func (r *photoRepository) DeleteAll(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&domain.Photo{})
	return result.RowsAffected, result.Error
}
