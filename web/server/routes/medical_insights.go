package routes

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"mqtt-streaming-server/domain"
)

type MedicalInsights struct {
	LastMonthCount             int `json:"last_month_count"`
	ExpiringNextMonthPeople    int `json:"expiring_next_month_people"`
	ExpiringNextMonthNames     []string `json:"expiring_next_month_names"`
	ExpiringNextMonthEntries   []ExpiringPerson `json:"expiring_next_month_entries"`
	TotalMedicinaMunciiInDocs  int `json:"total_medicina_muncii_in_documents"`
}

type ExpiringPerson struct {
	Name           string `json:"name"`
	ExpirationDate string `json:"expiration_date"`
}

func (ctlr PhotoController) GetMedicalInsights(w http.ResponseWriter, r *http.Request) {
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

	photos, err := ctlr.PhotoRepository.GetPhotos(ctx, filters)
	if err != nil {
		http.Error(w, "Failed to fetch medical insights", http.StatusInternalServerError)
		return
	}

	insights := calculateMedicalInsights(photos)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

func calculateMedicalInsights(photos []*domain.Photo) MedicalInsights {
	now := time.Now().UTC()
	lastMonthStart := now.AddDate(0, 0, -30)

	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	nextNextMonthStart := nextMonthStart.AddDate(0, 1, 0)

	insights := MedicalInsights{}
	expiringPeople := make(map[string]ExpiringPerson)

	for _, photo := range photos {
		if !isMedicinaMunciiDocument(photo) {
			continue
		}

		insights.TotalMedicinaMunciiInDocs++

		if !photo.Timestamp.Before(lastMonthStart) && !photo.Timestamp.After(now) {
			insights.LastMonthCount++
		}

		expiry := photo.MedicalData.DataUrmExaminari.Value
		if expiry.IsZero() {
			continue
		}

		expiry = expiry.UTC()
		if !expiry.Before(nextMonthStart) && expiry.Before(nextNextMonthStart) {
			key := personIdentityKey(photo)
			entry := ExpiringPerson{
				Name:           personDisplayName(photo),
				ExpirationDate: expiry.Format("2006-01-02"),
			}

			existing, exists := expiringPeople[key]
			if !exists || entry.ExpirationDate < existing.ExpirationDate {
				expiringPeople[key] = entry
			}
		}
	}

	insights.ExpiringNextMonthPeople = len(expiringPeople)
	insights.ExpiringNextMonthEntries = make([]ExpiringPerson, 0, len(expiringPeople))
	insights.ExpiringNextMonthNames = make([]string, 0, len(expiringPeople))
	for _, person := range expiringPeople {
		insights.ExpiringNextMonthEntries = append(insights.ExpiringNextMonthEntries, person)
		insights.ExpiringNextMonthNames = append(insights.ExpiringNextMonthNames, person.Name)
	}
	sort.Slice(insights.ExpiringNextMonthEntries, func(i, j int) bool {
		if insights.ExpiringNextMonthEntries[i].ExpirationDate == insights.ExpiringNextMonthEntries[j].ExpirationDate {
			return insights.ExpiringNextMonthEntries[i].Name < insights.ExpiringNextMonthEntries[j].Name
		}
		return insights.ExpiringNextMonthEntries[i].ExpirationDate < insights.ExpiringNextMonthEntries[j].ExpirationDate
	})
	sort.Strings(insights.ExpiringNextMonthNames)
	return insights
}

func isMedicinaMunciiDocument(photo *domain.Photo) bool {
	if photo == nil {
		return false
	}

	if !photo.MedicalData.Data.Value.IsZero() || !photo.MedicalData.DataUrmExaminari.Value.IsZero() {
		return true
	}
	if strings.TrimSpace(photo.MedicalData.TipControl.Value) != "" {
		return true
	}
	if strings.TrimSpace(photo.MedicalData.AvizMedical.Value) != "" {
		return true
	}
	if strings.TrimSpace(photo.MedicalData.CNP.Value) != "" {
		return true
	}

	return false
}

func personIdentityKey(photo *domain.Photo) string {
	cnp := strings.TrimSpace(photo.MedicalData.CNP.Value)
	if cnp != "" {
		return "cnp:" + cnp
	}

	nume := strings.TrimSpace(photo.MedicalData.Nume.Value)
	prenume := strings.TrimSpace(photo.MedicalData.Prenume.Value)
	if nume != "" || prenume != "" {
		return "name:" + strings.ToLower(nume+"|"+prenume)
	}

	return "doc:" + photo.ID
}

func personDisplayName(photo *domain.Photo) string {
	prenume := strings.TrimSpace(photo.MedicalData.Prenume.Value)
	nume := strings.TrimSpace(photo.MedicalData.Nume.Value)
	if prenume != "" || nume != "" {
		return strings.TrimSpace(prenume + " " + nume)
	}

	cnp := strings.TrimSpace(photo.MedicalData.CNP.Value)
	if len(cnp) >= 3 {
		return "Person CNP ending in " + cnp[len(cnp)-3:]
	}

	return "Unknown person"
}
