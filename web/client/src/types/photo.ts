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
export interface Photo {
  id: string;
  timestamp: string;
  image_type: string;
  processing_latency_ms: number;
  ocr_success: boolean;
  presigned_url: string;
  device_id: string;
  user_email: string;
  text: string;
  medical_data: MedicalData;
}

export interface PerformanceMetrics {
  total_documents: number;
  ocr_success_count: number;
  ocr_success_rate: number;
  average_latency_ms: number;
  p95_latency_ms: number;
}

export interface MedicalInsights {
  last_month_count: number;
  expiring_next_month_people: number;
  expiring_next_month_names: string[];
  expiring_next_month_entries: ExpiringPerson[];
  total_medicina_muncii_in_documents: number;
}

export interface ExpiringPerson {
  name: string;
  expiration_date: string;
}

export interface ProfessionFitEntry {
  profession: string;
  total: number;
  fit_count: number;
  fit_rate: number;
}

export interface FitByProfessionReport {
  total_people: number;
  total_fit_people: number;
  professors_total: number;
  professors_fit: number;
  by_profession: ProfessionFitEntry[];
}

export interface OverdueEntry {
  name: string;
  profession: string;
  expiration_date: string;
  days_overdue: number;
}

export interface OverdueReport {
  total_overdue_people: number;
  entries: OverdueEntry[];
}

export interface MonthlyCompliancePoint {
  month: string;
  completed_documents: number;
  fit_documents: number;
  overdue_documents: number;
  expiring_next_month_documents: number;
}

export interface MonthlyComplianceTrendReport {
  points: MonthlyCompliancePoint[];
}
