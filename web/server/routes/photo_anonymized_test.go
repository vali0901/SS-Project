package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
	"mqtt-streaming-server/routes"
)

func TestPhotoController_DownloadAnonymizedDataset(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	now := time.Now().UTC().Truncate(time.Second)
	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return([]*domain.Photo{
		{
			ID:        "photo-uuid-1",
			Timestamp: now,
			ImageType: "jpeg",
			UserEmail: "secret@example.com",
			DeviceID:  "device-secret",
			Text:      "Raw OCR text with personal details",
			MedicalData: domain.MedicalData{
				Nume:    domain.ExtractedField[string]{Value: "DOE"},
				Prenume: domain.ExtractedField[string]{Value: "JOHN"},
				CNP:     domain.ExtractedField[string]{Value: "1990101223344"},
				AvizMedical: domain.ExtractedField[string]{
					Value: "APT",
				},
				ControlPeriodic: domain.ExtractedField[bool]{Value: true},
			},
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/photos/anonymized/export", nil)
	ctx := context.WithValue(req.Context(), "role", "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.DownloadAnonymizedDataset(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if !strings.Contains(rr.Header().Get("Content-Disposition"), "attachment;") {
		t.Fatalf("expected attachment content disposition, got %q", rr.Header().Get("Content-Disposition"))
	}

	body := rr.Body.String()
	if strings.Contains(body, "secret@example.com") || strings.Contains(body, "1990101223344") || strings.Contains(body, "JOHN") {
		t.Fatalf("response contains PHI fields: %s", body)
	}

	var dataset routes.AnonymizedDataset
	if err := json.Unmarshal(rr.Body.Bytes(), &dataset); err != nil {
		t.Fatalf("failed to decode dataset JSON: %v", err)
	}
	if dataset.Count != 1 {
		t.Fatalf("expected count 1, got %d", dataset.Count)
	}
	if len(dataset.Documents) != 1 || dataset.Documents[0].DocumentID != "photo-uuid-1" {
		t.Fatalf("unexpected documents payload: %+v", dataset.Documents)
	}
}

func TestPhotoController_DownloadAnonymizedDataset_NonAdmin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	req := httptest.NewRequest(http.MethodGet, "/photos/anonymized/export", nil)
	ctx := context.WithValue(req.Context(), "role", "user")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.DownloadAnonymizedDataset(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestPhotoController_DownloadAnonymizedDataset_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return(nil, errors.New("db fail"))

	req := httptest.NewRequest(http.MethodGet, "/photos/anonymized/export", nil)
	ctx := context.WithValue(req.Context(), "role", "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.DownloadAnonymizedDataset(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
