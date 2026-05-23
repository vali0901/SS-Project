package routes

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"mqtt-streaming-server/domain"
)

type PerformanceMetrics struct {
	TotalDocuments         int     `json:"total_documents"`
	OCRSuccessCount        int     `json:"ocr_success_count"`
	OCRSuccessRate         float64 `json:"ocr_success_rate"`
	AverageLatencyMs       float64 `json:"average_latency_ms"`
	P95LatencyMs           int64   `json:"p95_latency_ms"`
}

func (ctlr PhotoController) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	filters := make(map[string]any)

	role, _ := ctx.Value("role").(string)
	if role != "admin" {
		email, _ := ctx.Value("email").(string)
		filters["user_email"] = email
	}

	start := r.URL.Query().Get("start")
	if start != "" {
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			http.Error(w, "Invalid start timestamp", http.StatusBadRequest)
			return
		}
		filters["start_date"] = time.Unix(startInt, 0)
	}

	end := r.URL.Query().Get("end")
	if end != "" {
		endInt, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			http.Error(w, "Invalid end timestamp", http.StatusBadRequest)
			return
		}
		filters["end_date"] = time.Unix(endInt, 0)
	}

	photos, err := ctlr.PhotoRepository.GetPhotos(ctx, filters)
	if err != nil {
		http.Error(w, "Failed to fetch performance metrics", http.StatusInternalServerError)
		return
	}

	metrics := calculatePerformanceMetrics(photos)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func calculatePerformanceMetrics(photos []*domain.Photo) PerformanceMetrics {
	total := len(photos)
	if total == 0 {
		return PerformanceMetrics{}
	}

	ocrSuccessCount := 0
	var totalLatency int64
	latencies := make([]int64, 0, total)

	for _, photo := range photos {
		legacyOCRSuccess := strings.TrimSpace(photo.Text) != "" && photo.Text != "OCR failed" && photo.Text != "OCR skipped"
		if photo.OCRSuccess || legacyOCRSuccess {
			ocrSuccessCount++
		}
		if photo.ProcessingLatencyMs > 0 {
			totalLatency += photo.ProcessingLatencyMs
			latencies = append(latencies, photo.ProcessingLatencyMs)
		}
	}

	averageLatency := 0.0
	p95Latency := int64(0)
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})
		p95Index := int(float64(len(latencies)-1) * 0.95)
		averageLatency = float64(totalLatency) / float64(len(latencies))
		p95Latency = latencies[p95Index]
	}

	return PerformanceMetrics{
		TotalDocuments:   total,
		OCRSuccessCount:  ocrSuccessCount,
		OCRSuccessRate:   (float64(ocrSuccessCount) / float64(total)) * 100,
		AverageLatencyMs: averageLatency,
		P95LatencyMs:     p95Latency,
	}
}
