package domain

import "time"

// ExtractedField este un wrapper generic pentru a stoca valoarea și scorul de încredere (0.0 - 1.0)
type ExtractedField[T any] struct {
	Value       T       `json:"value"`
	Confidence  float64 `json:"confidence"`
	IsEdited    bool    `json:"is_edited"`
	IsValidated bool    `json:"is_validated"`
}

// MedicalData reprezinta datele structurate extrase din fisa de aptitudine medicala
type MedicalData struct {
	// Header - Unitatea Medicala
	UnitateMedicala        ExtractedField[string] `json:"unitate_medicala"`
	AdresaUnitateMedicala  ExtractedField[string] `json:"adresa_unitate_medicala"`
	TelefonUnitateMedicala ExtractedField[string] `json:"telefon_unitate_medicala"`

	// Header - Tip Fisa
	NumarFisa ExtractedField[string] `json:"numar_fisa"`

	// Sectiune Angajator / Institutie
	SocietateUnitate ExtractedField[string] `json:"societate_unitate"`
	AdresaAngajator  ExtractedField[string] `json:"adresa_angajator"`
	TelefonAngajator ExtractedField[string] `json:"telefon_angajator"`

	// Date Personale Angajat
	Nume    ExtractedField[string] `json:"nume"`
	Prenume ExtractedField[string] `json:"prenume"`
	CNP     ExtractedField[string] `json:"cnp"`

	// Date Profesionale
	ProfesieFunctie ExtractedField[string] `json:"profesie_functie"`
	LocDeMunca      ExtractedField[string] `json:"loc_de_munca"`

	// Date Medicale
	TipControl          ExtractedField[string] `json:"tip_control"`
	ControlAngajare     ExtractedField[bool]   `json:"control_angajare"`
	ControlPeriodic     ExtractedField[bool]   `json:"control_periodic"`
	ControlAdaptare     ExtractedField[bool]   `json:"control_adaptare"`
	ControlReluare      ExtractedField[bool]   `json:"control_reluare"`
	ControlSupraveghere ExtractedField[bool]   `json:"control_supraveghere"`
	ControlAlte         ExtractedField[bool]   `json:"control_alte"`

	AvizMedical        ExtractedField[string] `json:"aviz_medical"`
	AvizApt            ExtractedField[bool]   `json:"aviz_apt"`
	AvizAptConditionat ExtractedField[bool]   `json:"aviz_apt_conditionat"`
	AvizInaptTemporar  ExtractedField[bool]   `json:"aviz_inapt_temporar"`
	AvizInapt          ExtractedField[bool]   `json:"aviz_inapt"`

	Recomandari      ExtractedField[string]    `json:"recomandari"`
	Data             ExtractedField[time.Time] `json:"data"`
	DataUrmExaminari ExtractedField[time.Time] `json:"data_urm_examinari"`
}
