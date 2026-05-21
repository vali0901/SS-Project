package domain

import "time"

// ExtractedField este un wrapper generic pentru a stoca valoarea și scorul de încredere (0.0 - 1.0)
type ExtractedField[T any] struct {
	Value      T       `json:"value" bson:"value"`
	Confidence float64 `json:"confidence" bson:"confidence"`
}

// MedicalData reprezinta datele structurate extrase din fisa de aptitudine medicala
type MedicalData struct {
	// Header - Unitatea Medicala
	UnitateMedicala        ExtractedField[string] `json:"unitate_medicala" bson:"unitate_medicala"`
	AdresaUnitateMedicala  ExtractedField[string] `json:"adresa_unitate_medicala" bson:"adresa_unitate_medicala"`
	TelefonUnitateMedicala ExtractedField[string] `json:"telefon_unitate_medicala" bson:"telefon_unitate_medicala"`

	// Header - Tip Fisa
	NumarFisa ExtractedField[string] `json:"numar_fisa" bson:"numar_fisa"`

	// Sectiune Angajator / Institutie
	SocietateUnitate ExtractedField[string] `json:"societate_unitate" bson:"societate_unitate"`
	AdresaAngajator  ExtractedField[string] `json:"adresa_angajator" bson:"adresa_angajator"`
	TelefonAngajator ExtractedField[string] `json:"telefon_angajator" bson:"telefon_angajator"`

	// Date Personale Angajat
	Nume    ExtractedField[string] `json:"nume" bson:"nume"`
	Prenume ExtractedField[string] `json:"prenume" bson:"prenume"`
	CNP     ExtractedField[string] `json:"cnp" bson:"cnp"`

	// Date Profesionale
	ProfesieFunctie ExtractedField[string] `json:"profesie_functie" bson:"profesie_functie"`
	LocDeMunca      ExtractedField[string] `json:"loc_de_munca" bson:"loc_de_munca"`

	// Date Medicale
	TipControl          ExtractedField[string] `json:"tip_control" bson:"tip_control"`
	ControlAngajare     ExtractedField[bool]   `json:"control_angajare" bson:"control_angajare"`
	ControlPeriodic     ExtractedField[bool]   `json:"control_periodic" bson:"control_periodic"`
	ControlAdaptare     ExtractedField[bool]   `json:"control_adaptare" bson:"control_adaptare"`
	ControlReluare      ExtractedField[bool]   `json:"control_reluare" bson:"control_reluare"`
	ControlSupraveghere ExtractedField[bool]   `json:"control_supraveghere" bson:"control_supraveghere"`
	ControlAlte         ExtractedField[bool]   `json:"control_alte" bson:"control_alte"`

	AvizMedical        ExtractedField[string] `json:"aviz_medical" bson:"aviz_medical"`
	AvizApt            ExtractedField[bool]   `json:"aviz_apt" bson:"aviz_apt"`
	AvizAptConditionat ExtractedField[bool]   `json:"aviz_apt_conditionat" bson:"aviz_apt_conditionat"`
	AvizInaptTemporar  ExtractedField[bool]   `json:"aviz_inapt_temporar" bson:"aviz_inapt_temporar"`
	AvizInapt          ExtractedField[bool]   `json:"aviz_inapt" bson:"aviz_inapt"`

	Recomandari      ExtractedField[string]    `json:"recomandari" bson:"recomandari"`
	Data             ExtractedField[time.Time] `json:"data" bson:"data"`
	DataUrmExaminari ExtractedField[time.Time] `json:"data_urm_examinari" bson:"data_urm_examinari"`
}
