package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/otiai10/gosseract/v2"
	"gorm.io/gorm"

	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/repository"
	"mqtt-streaming-server/utils"
)

type PhotoController struct {
	PhotoRepository domain.PhotoRepository
	ocrClient        *gosseract.Client
}

func InitPhotoRoutes(db *gorm.DB, ocrClient *gosseract.Client, mux *http.ServeMux) {
	photoController := &PhotoController{
		PhotoRepository: repository.NewPhotoRepository(db),
		ocrClient:       ocrClient,
	}

	mux.Handle("/photos", withAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			photoController.GetPhotos(w, r)
		} else if r.Method == http.MethodPost {
			photoController.UploadPhoto(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))
	mux.Handle("/photos/performance", withAuth(http.HandlerFunc(photoController.GetPerformanceMetrics)))
	mux.Handle("/photos/medical-insights", withAuth(http.HandlerFunc(photoController.GetMedicalInsights)))
	mux.Handle("/photos/anonymized/export", withAuth(http.HandlerFunc(photoController.DownloadAnonymizedDataset)))
	mux.Handle("/photos/all", withAuth(http.HandlerFunc(photoController.DeleteAllPhotos)))
	mux.Handle("/photos/", withAuth(http.HandlerFunc(photoController.HandlePhotoByID)))
}

func (ctlr PhotoController) HandlePhotoByID(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		ctlr.DeletePhoto(w, r)
		return
	} else if r.Method == http.MethodPatch || r.Method == http.MethodPut {
		ctlr.UpdatePhoto(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (ctlr PhotoController) GetPhotos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	text := r.URL.Query().Get("text")
	deviceID := r.URL.Query().Get("device_id")
	userEmailParam := r.URL.Query().Get("user_email")

	filters := make(map[string]any)

	// If user is not admin, only show their photos
	role, _ := ctx.Value("role").(string)
	if role != "admin" {
		email, _ := ctx.Value("email").(string)
		filters["user_email"] = email
	} else if userEmailParam != "" && userEmailParam != "all" {
		// Admin can filter by specific user
		filters["user_email"] = userEmailParam
	}

	if start != "" {
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err == nil {
			filters["start_date"] = time.Unix(startInt, 0)
		}
	}

	if end != "" {
		endInt, err := strconv.ParseInt(end, 10, 64)
		if err == nil {
			filters["end_date"] = time.Unix(endInt, 0)
		}
	}

	if text != "" {
		filters["text"] = text
	}

	if deviceID != "" {
		filters["device_id"] = deviceID
	}

	photos, err := ctlr.PhotoRepository.GetPhotos(ctx, filters)
	if err != nil {
		fmt.Println("Error fetching photos:", err)
		http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
		return
	}

	for _, photo := range photos {
		keyName := utils.PhotoStorageKey(photo.ID)
		photo.PresignedURL = utils.GetLocalURL(keyName)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(photos)
}

func (ctlr PhotoController) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	startProcessing := time.Now().UTC()
	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Photo field missing", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read file bytes
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Detect image type
	_, imageType, err := image.DecodeConfig(bytes.NewReader(fileBytes))
	if err != nil {
		// Fallback to extension if decoding fails
		ext := filepath.Ext(header.Filename)
		if ext != "" {
			imageType = strings.TrimPrefix(ext, ".")
		} else {
			imageType = "jpeg"
		}
	}

	// OCR Extraction
	text := "OCR skipped"
	ocrSuccess := false
	if ctlr.ocrClient != nil {
		ctlr.ocrClient.SetImageFromBytes(fileBytes)
		extracted, err := ctlr.ocrClient.Text()
		if err == nil {
			text = extracted
			ocrSuccess = strings.TrimSpace(extracted) != ""
		}
	}

	// Medical Data Extraction
	var medicalData *domain.MedicalData
	if utils.IsMedicalCertificate(text) {
		medicalData = utils.ParseMedicalCertificate(text)
	}

	timestamp := time.Now().UTC()
	deviceID := r.FormValue("device_id")
	if deviceID == "" {
		deviceID = "web_upload"
	}

	// Create photo object
	photo := &domain.Photo{
		ID:                  uuid.New().String(),
		Timestamp:           timestamp,
		ImageType:           imageType,
		ProcessingLatencyMs: time.Since(startProcessing).Milliseconds(),
		OCRSuccess:          ocrSuccess,
		DeviceID:            deviceID,
		UserEmail:           userEmail,
		Text:                text,
	}

	if medicalData != nil {
		photo.MedicalData = *medicalData
	}
	photoKey := utils.PhotoStorageKey(photo.ID)

	// Save to DB
	err = ctlr.PhotoRepository.Save(ctx, photo)
	if err != nil {
		http.Error(w, "Failed to save photo metadata", http.StatusInternalServerError)
		return
	}

	// Save file locally
	if err := utils.SaveToLocal(fileBytes, photoKey); err != nil {
		// We already saved to DB, so this is bad. 
		// In a real app we'd use a transaction or clean up.
		fmt.Printf("Failed to save photo file: %v\n", err)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(photo)
}

func (ctlr PhotoController) UpdatePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)
	role, _ := ctx.Value("role").(string)

	path := strings.TrimPrefix(r.URL.Path, "/photos/")
	if path == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}
	photoID := path

	// Check ownership
	photo, err := ctlr.PhotoRepository.GetByID(ctx, photoID)
	if err != nil {
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	if role != "admin" && photo.UserEmail != userEmail {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// We decode into domain.Photo to leverage json:",inline" and embedded MedicalData
	var updatedPhoto domain.Photo
	if err := json.NewDecoder(r.Body).Decode(&updatedPhoto); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Prepare update map for GORM
	update := map[string]any{
		"medical_data": updatedPhoto.MedicalData,
	}
	
	// If text was provided, update it too
	if updatedPhoto.Text != "" {
		update["text"] = updatedPhoto.Text
	}

	err = ctlr.PhotoRepository.Update(ctx, photoID, update)
	if err != nil {
		fmt.Println("Error updating photo:", err)
		http.Error(w, "Failed to update photo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Photo updated successfully"})
}

func (ctlr PhotoController) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userEmail, _ := ctx.Value("email").(string)
	role, _ := ctx.Value("role").(string)

	// Extract photo ID from URL path: /photos/{id}
	path := strings.TrimPrefix(r.URL.Path, "/photos/")
	if path == "" {
		http.Error(w, "Photo ID required", http.StatusBadRequest)
		return
	}
	photoID := path

	// Get the photo to find the file name and check ownership
	photo, err := ctlr.PhotoRepository.GetByID(ctx, photoID)
	if err != nil {
		fmt.Println("Error getting photo:", err)
		http.Error(w, "Photo not found", http.StatusNotFound)
		return
	}

	if role != "admin" && photo.UserEmail != userEmail {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}
	fileName := utils.PhotoStorageKey(photo.ID)

	// Delete from database
	err = ctlr.PhotoRepository.Delete(ctx, photoID)
	if err != nil {
		fmt.Println("Error deleting photo:", err)
		http.Error(w, "Failed to delete photo", http.StatusInternalServerError)
		return
	}

	// Delete the image file from local storage
	if err := os.Remove(filepath.Join("uploads", fileName)); err != nil {
		fmt.Printf("Warning: Could not delete file %s: %v\n", fileName, err)
		// Don't fail the request - the DB record is already deleted
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Photo deleted successfully"})
}

func (ctlr PhotoController) DeleteAllPhotos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	role, _ := ctx.Value("role").(string)

	if role != "admin" {
		http.Error(w, "Unauthorized: Admin only", http.StatusForbidden)
		return
	}

	// Delete all photos from database
	deletedCount, err := ctlr.PhotoRepository.DeleteAll(ctx)
	if err != nil {
		fmt.Println("Error deleting all photos:", err)
		http.Error(w, "Failed to delete photos", http.StatusInternalServerError)
		return
	}

	// Delete all image files from uploads/photos directory
	photosDir := "uploads/photos"
	files, err := filepath.Glob(filepath.Join(photosDir, "*"))
	if err == nil {
		for _, f := range files {
			os.Remove(f)
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "All photos deleted successfully",
		"deleted": deletedCount,
	})
}
