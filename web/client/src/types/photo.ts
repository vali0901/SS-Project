export interface ExtractedField<T> {
  value: T;
  confidence: number;
  is_edited?: boolean;
  is_validated?: boolean;
}

export interface MedicalData {
  unitate_medicala: ExtractedField<string>;
  adresa_unitate_medicala: ExtractedField<string>;
  telefon_unitate_medicala: ExtractedField<string>;
  numar_fisa: ExtractedField<string>;
  societate_unitate: ExtractedField<string>;
  adresa_angajator: ExtractedField<string>;
  telefon_angajator: ExtractedField<string>;
  nume: ExtractedField<string>;
  prenume: ExtractedField<string>;
  cnp: ExtractedField<string>;
  profesie_functie: ExtractedField<string>;
  loc_de_munca: ExtractedField<string>;
  tip_control: ExtractedField<string>;
  control_angajare: ExtractedField<boolean>;
  control_periodic: ExtractedField<boolean>;
  control_adaptare: ExtractedField<boolean>;
  control_reluare: ExtractedField<boolean>;
  control_supraveghere: ExtractedField<boolean>;
  control_alte: ExtractedField<boolean>;
  aviz_medical: ExtractedField<string>;
  aviz_apt: ExtractedField<boolean>;
  aviz_apt_conditionat: ExtractedField<boolean>;
  aviz_inapt_temporar: ExtractedField<boolean>;
  aviz_inapt: ExtractedField<boolean>;
  recomandari: ExtractedField<string>;
  data: ExtractedField<string>;
  data_urm_examinari: ExtractedField<string>;
}

// Interface for photo data
export interface Photo extends MedicalData {
  id: string;
  timestamp: string;
  image_type: string;
  presigned_url: string;
  device_id: string;
  text: string;
}
