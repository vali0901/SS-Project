package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
	"mqtt-streaming-server/routes"
)

func TestPhotoController_GetPerformanceMetrics_Admin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return([]*domain.Photo{
		{OCRSuccess: true, ProcessingLatencyMs: 120},
		{OCRSuccess: false, ProcessingLatencyMs: 300},
		{OCRSuccess: true, ProcessingLatencyMs: 180},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/photos/performance?start=100&end=200", nil)
	ctx := context.WithValue(req.Context(), "role", "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.GetPerformanceMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var metrics routes.PerformanceMetrics
	if err := json.Unmarshal(rr.Body.Bytes(), &metrics); err != nil {
		t.Fatalf("failed to decode metrics: %v", err)
	}

	if metrics.TotalDocuments != 3 {
		t.Fatalf("expected total 3, got %d", metrics.TotalDocuments)
	}
	if metrics.OCRSuccessCount != 2 {
		t.Fatalf("expected OCR success count 2, got %d", metrics.OCRSuccessCount)
	}
	if metrics.P95LatencyMs != 180 {
		t.Fatalf("expected p95 latency 180ms, got %d", metrics.P95LatencyMs)
	}
}

func TestPhotoController_GetPerformanceMetrics_UserScoped(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, filters map[string]any) ([]*domain.Photo, error) {
			if filters["user_email"] != "user@example.com" {
				t.Fatalf("expected user_email filter to be applied, got %#v", filters["user_email"])
			}
			return []*domain.Photo{}, nil
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/photos/performance", nil)
	ctx := context.WithValue(req.Context(), "role", "user")
	ctx = context.WithValue(ctx, "email", "user@example.com")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.GetPerformanceMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestPhotoController_GetPerformanceMetrics_InvalidStart(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	req := httptest.NewRequest(http.MethodGet, "/photos/performance?start=bad", nil)
	ctx := context.WithValue(req.Context(), "role", "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.GetPerformanceMetrics(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
