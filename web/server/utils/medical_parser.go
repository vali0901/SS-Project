package utils

import (
	"regexp"
	"strings"
	"time"

	"mqtt-streaming-server/domain"
)

// IsMedicalCertificate checks if the OCR text likely belongs to a medical certificate
func IsMedicalCertificate(ocrText string) bool {
	if ocrText == "" || strings.Contains(ocrText, "OCR failed") {
		return false
	}
	keywords := []string{"FISA DE APTITUDINE", "UNITATEA MEDICALA", "CNP", "AVIZ MEDICAL", "ANGAJARE", "CONTROL PERIODIC"}
	count := 0
	upperText := strings.ToUpper(ocrText)
	for _, kw := range keywords {
		if strings.Contains(upperText, kw) {
			count++
		}
	}
	return count >= 1
}

// ParseMedicalCertificate extrage datele structurate si scorurile de incredere
func ParseMedicalCertificate(ocrText string) *domain.MedicalData {
	if ocrText == "" || strings.Contains(ocrText, "OCR failed") {
		return nil
	}

	data := &domain.MedicalData{}
	cleanText := normalizeOCRText(ocrText)

	// --- Header Section ---
	data.UnitateMedicala = extractStringField(cleanText, `(?i)UNITATEA\s+MEDICALA:\s*(.*?)(?:\s+UNITATEA|\s+ADRESA|\[|$)`, 0.9)
	data.AdresaUnitateMedicala = extractStringField(cleanText, `(?i)ADRESA:\s*(.*?)(?:\s+ADRESA|\s+DEPARTAMENT|\s+TEL|$)`, 0.9)

	data.TelefonUnitateMedicala = extractStringField(cleanText, `(?i)TEL[:\+0-9\s,]+(?:FAX[:\+0-9\s,]+)?(.*?)(?:\s+TEL|BUCURESTI|$)`, 0.9)
	if data.TelefonUnitateMedicala.Value == "" {
		// Scadere scor pentru fallback
		data.TelefonUnitateMedicala = extractStringField(cleanText, `(?i)(TEL[:;\.\s]*\+?\s*\d[\d\s\-\.]{8,})`, 0.7)
	}

	data.NumarFisa = extractStringField(cleanText, `(?i)FISA\s+DE\s+APTITUDINE\s+NR[\.:]?\s*(\d+)`, 0.95)

	// --- Employer Section ---
	data.SocietateUnitate = extractStringField(cleanText, `(?i)Soci[ec]tate,?\s*unitate,?\s*(?:etc[\.:]?)?\s*:\s*(.*?)(?:\s+Societate|\s+Adresa|$)`, 0.9)
	if data.SocietateUnitate.Value == "" {
		data.SocietateUnitate = extractStringField(cleanText, `(?i)(UNIVERSITATEA\s+(?:NATIONALA\s+DE\s+STIINTA\s+SI\s+TEHNOLOGIE\s+)?POLITEHNICA\s+(?:DIN\s+)?[A-Z]+)`, 0.75)
	}

	data.AdresaAngajator = extractStringField(cleanText[len(cleanText)/2:], `(?i)Adresa[:;]?\s*(.*?)(?:\s+Adresa|\s+Telefon|$)`, 0.85)

	// --- Personal Data ---
	data.Nume = extractStringField(cleanText, `(?i)NUME[:;]?\s*([A-Za-z\s\-]+?)(?:\s+NUME|\s+PRENUME|$)`, 0.95)
	data.Prenume = extractStringField(cleanText, `(?i)PRENUME[:;]?\s*([A-Za-z\s\-]+?)(?:\s+PRENUME|\s+CNP|$)`, 0.95)
	data.CNP = extractStringField(cleanText, `(?i)CNP[:;]?\s*(\d+)`, 0.98) // CNP e usor de validat

	// --- Professional Data ---
	data.ProfesieFunctie = extractStringField(cleanText, `(?i)Profesie\s*[\/\|]\s*functie[:;]?\s*(.*?)(?:\s+Rrofesle|\s+Locul|$)`, 0.9)
	data.LocDeMunca = extractStringField(cleanText, `(?i)Locul?\s+de\s+munca[:;]?\s*(.*?)(?:\s+Locul|\s+AVIZ|$)`, 0.9)

	// --- Medical Data (Checkboxes) ---
	data.ControlAngajare = detectCheckbox(cleanText, `Angajare`)
	data.ControlPeriodic = detectCheckbox(cleanText, `Control\s*(?:medical)?(?:periodic)?`)
	data.ControlAdaptare = detectCheckbox(cleanText, `Adaptare`)
	data.ControlReluare = detectCheckbox(cleanText, `R[eo]luar[ec]a`)
	data.ControlSupraveghere = detectCheckbox(cleanText, `Supraveghere`)
	data.ControlAlte = detectCheckbox(cleanText, `Alte|Ane`)

	// Calculam "TipControl" determinist pe baza celei mai sigure bife
	data.TipControl = deduceMainControl(data)

	// Aviz Medical
	data.AvizApt = detectCheckbox(cleanText, `APT\s*:`)
	data.AvizAptConditionat = detectCheckbox(cleanText, `APT\s+CONDITIONAT|ApTCONDITIONAT`)
	data.AvizInaptTemporar = detectCheckbox(cleanText, `INAPT\s+TEMPORAR`)
	data.AvizInapt = detectCheckbox(cleanText, `INAPT\s*:`)

	data.AvizMedical = deduceAvizMedical(data)

	// --- Dates ---
	data.Data = parseDateStr(extractFieldRaw(cleanText, `(?i)Data[:;\s\_]*(\d{2}[\.\/\-]\d{2}[\.\/\-]\d{4})`))
	data.DataUrmExaminari = parseDateStr(extractFieldRaw(cleanText, `(?i)Data\s+urmatoarei\s+examinari[:;\s\_]*(\d{2}[\.\/\-]\d{2}[\.\/\-]\d{4})`))

	return data
}

func normalizeOCRText(text string) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(text, " "))
}

// extractFieldRaw extrage strict string-ul pentru a fi procesat de alte functii (ex. date)
func extractFieldRaw(text, pattern string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(strings.TrimRight(matches[1], ",;:. "))
	}
	return ""
}

// extractStringField extrage un string si ataseaza o incredere bazata pe succesul operatiunii
func extractStringField(text, pattern string, baseConfidence float64) domain.ExtractedField[string] {
	val := extractFieldRaw(text, pattern)
	conf := 0.0
	if val != "" {
		conf = baseConfidence
		// Penalizare minora daca string-ul contine caractere OCR suspecte (ex. multiple puncte sau underscore)
		if strings.Contains(val, "..") || strings.Contains(val, "__") {
			conf -= 0.2
		}
	}
	return domain.ExtractedField[string]{Value: val, Confidence: conf}
}

// detectCheckbox returneaza un boolean cu scor de incredere bazat pe claritatea bifei
func detectCheckbox(text, labelPattern string) domain.ExtractedField[bool] {
	re := regexp.MustCompile(`(?i)` + labelPattern + `\s*([^a-zA-Z]{1,10})`)
	matches := re.FindAllStringSubmatch(text, -1)

	if len(matches) == 0 {
		// Nu am gasit label-ul. Nu stim starea.
		return domain.ExtractedField[bool]{Value: false, Confidence: 0.0}
	}

	for _, match := range matches {
		if len(match) > 1 {
			gap := match[1]

			// Casetă perfect vizibilă, clar goală
			if strings.Contains(gap, "[]") || strings.Contains(gap, "[ ]") {
				return domain.ExtractedField[bool]{Value: false, Confidence: 0.95}
			}

			// Casetă clar bifată (ex: [X], [v], [-])
			if regexp.MustCompile(`\[\s*[xXvV\-\_]\s*\]`).MatchString(gap) {
				return domain.ExtractedField[bool]{Value: true, Confidence: 0.95}
			}

			// Marker corupt de OCR (dar pare o bifare, ex: [[], sau [x)
			if regexp.MustCompile(`\[[xXvV\-\_\[]+\]?`).MatchString(gap) {
				return domain.ExtractedField[bool]{Value: true, Confidence: 0.70}
			}

			// Alternativa corupta fara brackete (ex: APT: :)
			if strings.Contains(gap, ": :") || strings.Contains(gap, ": O") {
				return domain.ExtractedField[bool]{Value: true, Confidence: 0.60}
			}
		}
	}

	// S-a găsit eticheta, dar spațiul e neclar/zgomotos (fără o casetă evidentă)
	return domain.ExtractedField[bool]{Value: false, Confidence: 0.40}
}

// parseDateStr converteste un string murdar si atribuie incredere bazata pe reusita parsarii
func parseDateStr(dateStr string) domain.ExtractedField[time.Time] {
	if dateStr == "" {
		return domain.ExtractedField[time.Time]{Value: time.Time{}, Confidence: 0.0}
	}

	normalizedDate := strings.ReplaceAll(dateStr, ".", "/")
	normalizedDate = strings.ReplaceAll(normalizedDate, "-", "/")

	t, err := time.Parse("02/01/2006", normalizedDate)
	if err == nil {
		// Dacă a fost nevoie de normalizare, scădem foarte puțin din încredere
		conf := 0.95
		if normalizedDate != dateStr {
			conf = 0.85
		}
		return domain.ExtractedField[time.Time]{Value: t, Confidence: conf}
	}

	return domain.ExtractedField[time.Time]{Value: time.Time{}, Confidence: 0.0}
}

// deduceMainControl unifică logica pentru a determina Tipul Controlului cu încredere
func deduceMainControl(data *domain.MedicalData) domain.ExtractedField[string] {
	// Găsește bifa cu cea mai mare încredere care este true
	options := []struct {
		Label string
		Field domain.ExtractedField[bool]
	}{
		{"Angajare", data.ControlAngajare},
		{"Control medical periodic", data.ControlPeriodic},
		{"Adaptare", data.ControlAdaptare},
		{"Reluarea muncii", data.ControlReluare},
		{"Supraveghere speciala", data.ControlSupraveghere},
	}

	bestConf := 0.0
	bestVal := ""

	for _, opt := range options {
		if opt.Field.Value && opt.Field.Confidence > bestConf {
			bestConf = opt.Field.Confidence
			bestVal = opt.Label
		}
	}

	return domain.ExtractedField[string]{Value: bestVal, Confidence: bestConf}
}

// deduceAvizMedical face acelasi lucru pentru Avizul final
func deduceAvizMedical(data *domain.MedicalData) domain.ExtractedField[string] {
	options := []struct {
		Label string
		Field domain.ExtractedField[bool]
	}{
		{"APT CONDITIONAT", data.AvizAptConditionat},
		{"INAPT TEMPORAR", data.AvizInaptTemporar},
		{"INAPT", data.AvizInapt},
		{"APT", data.AvizApt},
	}

	bestConf := 0.0
	bestVal := ""

	for _, opt := range options {
		if opt.Field.Value && opt.Field.Confidence > bestConf {
			bestConf = opt.Field.Confidence
			bestVal = opt.Label
		}
	}

	return domain.ExtractedField[string]{Value: bestVal, Confidence: bestConf}
}
