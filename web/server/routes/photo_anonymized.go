package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"mqtt-streaming-server/domain"
)

// AnonymizedDocument intentionally excludes PHI such as names, CNP, user email,
// device identifiers, addresses, phone numbers, raw OCR text, and image URLs.
type AnonymizedDocument struct {
	DocumentID string                `json:"document_id"`
	Timestamp  time.Time             `json:"timestamp"`
	ImageType  string                `json:"image_type"`
	Medical    AnonymizedMedicalData `json:"medical_data"`
}

type AnonymizedMedicalData struct {
	TipControl          domain.ExtractedField[string]    `json:"tip_control"`
	ControlAngajare     domain.ExtractedField[bool]      `json:"control_angajare"`
	ControlPeriodic     domain.ExtractedField[bool]      `json:"control_periodic"`
	ControlAdaptare     domain.ExtractedField[bool]      `json:"control_adaptare"`
	ControlReluare      domain.ExtractedField[bool]      `json:"control_reluare"`
	ControlSupraveghere domain.ExtractedField[bool]      `json:"control_supraveghere"`
	ControlAlte         domain.ExtractedField[bool]      `json:"control_alte"`
	AvizMedical         domain.ExtractedField[string]    `json:"aviz_medical"`
	AvizApt             domain.ExtractedField[bool]      `json:"aviz_apt"`
	AvizAptConditionat  domain.ExtractedField[bool]      `json:"aviz_apt_conditionat"`
	AvizInaptTemporar   domain.ExtractedField[bool]      `json:"aviz_inapt_temporar"`
	AvizInapt           domain.ExtractedField[bool]      `json:"aviz_inapt"`
	Data                domain.ExtractedField[time.Time] `json:"data"`
	DataUrmExaminari    domain.ExtractedField[time.Time] `json:"data_urm_examinari"`
}

type AnonymizedDataset struct {
	GeneratedAt time.Time            `json:"generated_at"`
	Count       int                  `json:"count"`
	Documents   []AnonymizedDocument `json:"documents"`
}

func (ctlr PhotoController) DownloadAnonymizedDataset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	role, _ := ctx.Value("role").(string)
	if role != "admin" {
		http.Error(w, "Unauthorized: Admin only", http.StatusForbidden)
		return
	}

	filters := make(map[string]any)
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")

	if start != "" {
		startInt, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			http.Error(w, "Invalid start timestamp", http.StatusBadRequest)
			return
		}
		filters["start_date"] = time.Unix(startInt, 0)
	}

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
		http.Error(w, "Failed to fetch photos", http.StatusInternalServerError)
		return
	}

	docs := make([]AnonymizedDocument, 0, len(photos))
	for _, photo := range photos {
		docs = append(docs, anonymizePhoto(photo))
	}

	export := AnonymizedDataset{
		GeneratedAt: time.Now().UTC(),
		Count:       len(docs),
		Documents:   docs,
	}

	filename := fmt.Sprintf("anonymized_dataset_%d.json", time.Now().UTC().Unix())
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	if err := json.NewEncoder(w).Encode(export); err != nil {
		http.Error(w, "Failed to encode anonymized dataset", http.StatusInternalServerError)
		return
	}
}

func anonymizePhoto(photo *domain.Photo) AnonymizedDocument {
	return AnonymizedDocument{
		DocumentID: photo.ID,
		Timestamp:  photo.Timestamp,
		ImageType:  photo.ImageType,
		Medical: AnonymizedMedicalData{
			TipControl:          photo.MedicalData.TipControl,
			ControlAngajare:     photo.MedicalData.ControlAngajare,
			ControlPeriodic:     photo.MedicalData.ControlPeriodic,
			ControlAdaptare:     photo.MedicalData.ControlAdaptare,
			ControlReluare:      photo.MedicalData.ControlReluare,
			ControlSupraveghere: photo.MedicalData.ControlSupraveghere,
			ControlAlte:         photo.MedicalData.ControlAlte,
			AvizMedical:         photo.MedicalData.AvizMedical,
			AvizApt:             photo.MedicalData.AvizApt,
			AvizAptConditionat:  photo.MedicalData.AvizAptConditionat,
			AvizInaptTemporar:   photo.MedicalData.AvizInaptTemporar,
			AvizInapt:           photo.MedicalData.AvizInapt,
			Data:                photo.MedicalData.Data,
			DataUrmExaminari:    photo.MedicalData.DataUrmExaminari,
		},
	}
}
