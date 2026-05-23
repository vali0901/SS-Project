package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"mqtt-streaming-server/domain"
	mock_domain "mqtt-streaming-server/mocks"
	"mqtt-streaming-server/routes"
)

func TestPhotoController_GetMedicalInsights(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mock_domain.NewMockPhotoRepository(ctrl)
	ctlr := routes.PhotoController{PhotoRepository: mockRepo}

	now := time.Now().UTC()
	nextMonth := time.Date(now.Year(), now.Month()+1, 10, 0, 0, 0, 0, time.UTC)
	lastMonth := now.AddDate(0, 0, -10)

	mockRepo.EXPECT().GetPhotos(gomock.Any(), gomock.Any()).Return([]*domain.Photo{
		{
			ID:        "p1",
			Timestamp: lastMonth,
			MedicalData: domain.MedicalData{
				Nume:             domain.ExtractedField[string]{Value: "Popescu"},
				Prenume:          domain.ExtractedField[string]{Value: "Maria"},
				CNP:              domain.ExtractedField[string]{Value: "111"},
				TipControl:       domain.ExtractedField[string]{Value: "Control Periodic"},
				DataUrmExaminari: domain.ExtractedField[time.Time]{Value: nextMonth},
			},
		},
		{
			ID:        "p2",
			Timestamp: lastMonth,
			MedicalData: domain.MedicalData{
				CNP:              domain.ExtractedField[string]{Value: "111"},
				TipControl:       domain.ExtractedField[string]{Value: "Control Periodic"},
				DataUrmExaminari: domain.ExtractedField[time.Time]{Value: nextMonth.AddDate(0, 0, 1)},
			},
		},
		{
			ID:        "non-med",
			Timestamp: lastMonth,
		},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/photos/medical-insights", nil)
	ctx := context.WithValue(req.Context(), "role", "admin")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	ctlr.GetMedicalInsights(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp routes.MedicalInsights
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalMedicinaMunciiInDocs != 2 {
		t.Fatalf("expected total medicina muncii 2, got %d", resp.TotalMedicinaMunciiInDocs)
	}
	if resp.LastMonthCount != 2 {
		t.Fatalf("expected last month count 2, got %d", resp.LastMonthCount)
	}
	if resp.ExpiringNextMonthPeople != 1 {
		t.Fatalf("expected expiring next month people 1, got %d", resp.ExpiringNextMonthPeople)
	}
	if len(resp.ExpiringNextMonthNames) != 1 || resp.ExpiringNextMonthNames[0] != "Maria Popescu" {
		t.Fatalf("expected expiring name list to contain Maria Popescu, got %+v", resp.ExpiringNextMonthNames)
	}
}
