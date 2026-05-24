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

type ProfessionFitEntry struct {
	Profession string  `json:"profession"`
	Total      int     `json:"total"`
	FitCount   int     `json:"fit_count"`
	FitRate    float64 `json:"fit_rate"`
}

type FitByProfessionReport struct {
	TotalPeople     int                  `json:"total_people"`
	TotalFitPeople  int                  `json:"total_fit_people"`
	ProfessorsTotal int                  `json:"professors_total"`
	ProfessorsFit   int                  `json:"professors_fit"`
	ByProfession    []ProfessionFitEntry `json:"by_profession"`
}

type OverdueEntry struct {
	Name           string `json:"name"`
	Profession     string `json:"profession"`
	ExpirationDate string `json:"expiration_date"`
	DaysOverdue    int    `json:"days_overdue"`
}

type OverdueReport struct {
	TotalOverduePeople int            `json:"total_overdue_people"`
	Entries            []OverdueEntry `json:"entries"`
}

type MonthlyCompliancePoint struct {
	Month                      string `json:"month"`
	CompletedDocuments         int    `json:"completed_documents"`
	FitDocuments               int    `json:"fit_documents"`
	OverdueDocuments           int    `json:"overdue_documents"`
	ExpiringNextMonthDocuments int    `json:"expiring_next_month_documents"`
}

type MonthlyComplianceTrendReport struct {
	Points []MonthlyCompliancePoint `json:"points"`
}

func (ctlr PhotoController) GetFitByProfessionReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	photos, err := ctlr.getScopedPhotos(r)
	if err != nil {
		http.Error(w, "Failed to fetch report data", http.StatusInternalServerError)
		return
	}

	start, end, err := parseStartEndFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	report := calculateFitByProfessionReport(filterPhotosByTimestamp(photos, start, end))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (ctlr PhotoController) GetOverdueReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	photos, err := ctlr.getScopedPhotos(r)
	if err != nil {
		http.Error(w, "Failed to fetch report data", http.StatusInternalServerError)
		return
	}

	report := calculateOverdueReport(photos)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (ctlr PhotoController) GetMonthlyComplianceTrendReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	photos, err := ctlr.getScopedPhotos(r)
	if err != nil {
		http.Error(w, "Failed to fetch report data", http.StatusInternalServerError)
		return
	}

	start, end, err := parseStartEndFromQuery(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if start == nil || end == nil {
		now := time.Now().UTC()
		defaultStart := now.AddDate(0, -5, 0)
		s := time.Date(defaultStart.Year(), defaultStart.Month(), 1, 0, 0, 0, 0, time.UTC)
		e := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0).Add(-time.Nanosecond)
		start = &s
		end = &e
	}

	report := calculateMonthlyComplianceTrendReport(photos, *start, *end)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (ctlr PhotoController) getScopedPhotos(r *http.Request) ([]*domain.Photo, error) {
	ctx := r.Context()
	filters := make(map[string]any)

	role, _ := ctx.Value("role").(string)
	if role != "admin" {
		email, _ := ctx.Value("email").(string)
		filters["user_email"] = email
	}

	return ctlr.PhotoRepository.GetPhotos(ctx, filters)
}

func parseStartEndFromQuery(r *http.Request) (*time.Time, *time.Time, error) {
	var start *time.Time
	var end *time.Time

	if rawStart := r.URL.Query().Get("start"); rawStart != "" {
		startInt, err := strconv.ParseInt(rawStart, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		ts := time.Unix(startInt, 0).UTC()
		start = &ts
	}

	if rawEnd := r.URL.Query().Get("end"); rawEnd != "" {
		endInt, err := strconv.ParseInt(rawEnd, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		te := time.Unix(endInt, 0).UTC()
		end = &te
	}

	return start, end, nil
}

func filterPhotosByTimestamp(photos []*domain.Photo, start, end *time.Time) []*domain.Photo {
	if start == nil && end == nil {
		return photos
	}

	filtered := make([]*domain.Photo, 0, len(photos))
	for _, p := range photos {
		if p == nil {
			continue
		}
		if start != nil && p.Timestamp.Before(*start) {
			continue
		}
		if end != nil && p.Timestamp.After(*end) {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func calculateFitByProfessionReport(photos []*domain.Photo) FitByProfessionReport {
	latestByPerson := make(map[string]*domain.Photo)
	for _, p := range photos {
		if !isMedicinaMunciiDocument(p) {
			continue
		}
		key := personIdentityKey(p)
		existing, ok := latestByPerson[key]
		if !ok || p.Timestamp.After(existing.Timestamp) {
			latestByPerson[key] = p
		}
	}

	entriesMap := make(map[string]*ProfessionFitEntry)
	report := FitByProfessionReport{}

	for _, p := range latestByPerson {
		report.TotalPeople++
		profession := professionValue(p)
		if _, ok := entriesMap[profession]; !ok {
			entriesMap[profession] = &ProfessionFitEntry{Profession: profession}
		}
		entriesMap[profession].Total++

		if strings.Contains(strings.ToLower(profession), "profesor") {
			report.ProfessorsTotal++
		}

		if isFitStatus(p) {
			report.TotalFitPeople++
			entriesMap[profession].FitCount++
			if strings.Contains(strings.ToLower(profession), "profesor") {
				report.ProfessorsFit++
			}
		}
	}

	report.ByProfession = make([]ProfessionFitEntry, 0, len(entriesMap))
	for _, entry := range entriesMap {
		if entry.Total > 0 {
			entry.FitRate = (float64(entry.FitCount) / float64(entry.Total)) * 100
		}
		report.ByProfession = append(report.ByProfession, *entry)
	}

	sort.Slice(report.ByProfession, func(i, j int) bool {
		if report.ByProfession[i].Total == report.ByProfession[j].Total {
			return report.ByProfession[i].Profession < report.ByProfession[j].Profession
		}
		return report.ByProfession[i].Total > report.ByProfession[j].Total
	})

	return report
}

func calculateOverdueReport(photos []*domain.Photo) OverdueReport {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	latestByPerson := make(map[string]*domain.Photo)
	for _, p := range photos {
		if !isMedicinaMunciiDocument(p) || p.MedicalData.DataUrmExaminari.Value.IsZero() {
			continue
		}
		key := personIdentityKey(p)
		existing, ok := latestByPerson[key]
		if !ok || p.Timestamp.After(existing.Timestamp) {
			latestByPerson[key] = p
		}
	}

	report := OverdueReport{}
	for _, p := range latestByPerson {
		expiry := p.MedicalData.DataUrmExaminari.Value.UTC()
		if !expiry.Before(today) {
			continue
		}

		days := int(today.Sub(time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 0, 0, 0, 0, time.UTC)).Hours() / 24)
		report.Entries = append(report.Entries, OverdueEntry{
			Name:           personDisplayName(p),
			Profession:     professionValue(p),
			ExpirationDate: expiry.Format("2006-01-02"),
			DaysOverdue:    days,
		})
	}

	sort.Slice(report.Entries, func(i, j int) bool {
		if report.Entries[i].DaysOverdue == report.Entries[j].DaysOverdue {
			return report.Entries[i].Name < report.Entries[j].Name
		}
		return report.Entries[i].DaysOverdue > report.Entries[j].DaysOverdue
	})
	report.TotalOverduePeople = len(report.Entries)

	return report
}

func calculateMonthlyComplianceTrendReport(photos []*domain.Photo, start, end time.Time) MonthlyComplianceTrendReport {
	startMonth := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	if endMonth.Before(startMonth) {
		return MonthlyComplianceTrendReport{}
	}

	type bucket struct {
		month                      time.Time
		completedDocuments         int
		fitDocuments               int
		overdueDocuments           int
		expiringNextMonthDocuments int
	}

	buckets := make([]bucket, 0)
	monthIndex := make(map[string]int)
	for m := startMonth; !m.After(endMonth); m = m.AddDate(0, 1, 0) {
		key := m.Format("2006-01")
		monthIndex[key] = len(buckets)
		buckets = append(buckets, bucket{month: m})
	}

	for _, p := range photos {
		if p == nil || !isMedicinaMunciiDocument(p) {
			continue
		}

		if p.Timestamp.Before(start) || p.Timestamp.After(end) {
			continue
		}

		monthKey := time.Date(p.Timestamp.Year(), p.Timestamp.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
		idx, ok := monthIndex[monthKey]
		if !ok {
			continue
		}

		b := &buckets[idx]
		b.completedDocuments++
		if isFitStatus(p) {
			b.fitDocuments++
		}

		expiry := p.MedicalData.DataUrmExaminari.Value.UTC()
		if expiry.IsZero() {
			continue
		}

		monthEnd := b.month.AddDate(0, 1, 0).Add(-time.Nanosecond)
		if expiry.Before(monthEnd) {
			b.overdueDocuments++
		}

		nextMonthStart := time.Date(b.month.Year(), b.month.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		nextNextMonthStart := nextMonthStart.AddDate(0, 1, 0)
		if !expiry.Before(nextMonthStart) && expiry.Before(nextNextMonthStart) {
			b.expiringNextMonthDocuments++
		}
	}

	points := make([]MonthlyCompliancePoint, 0, len(buckets))
	for _, b := range buckets {
		points = append(points, MonthlyCompliancePoint{
			Month:                      b.month.Format("2006-01"),
			CompletedDocuments:         b.completedDocuments,
			FitDocuments:               b.fitDocuments,
			OverdueDocuments:           b.overdueDocuments,
			ExpiringNextMonthDocuments: b.expiringNextMonthDocuments,
		})
	}

	return MonthlyComplianceTrendReport{Points: points}
}

func professionValue(photo *domain.Photo) string {
	if photo == nil {
		return "Unknown"
	}

	profession := strings.TrimSpace(photo.MedicalData.ProfesieFunctie.Value)
	if profession != "" {
		return profession
	}

	workplace := strings.TrimSpace(photo.MedicalData.LocDeMunca.Value)
	if workplace != "" {
		return workplace
	}

	return "Unknown"
}

func isFitStatus(photo *domain.Photo) bool {
	if photo == nil {
		return false
	}

	if photo.MedicalData.AvizApt.Value {
		return true
	}

	aviz := strings.ToUpper(strings.TrimSpace(photo.MedicalData.AvizMedical.Value))
	return strings.HasPrefix(aviz, "APT")
}
