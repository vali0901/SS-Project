import React, { useState } from 'react';
import fallbackImage from '../../assets/photo-fallback.svg';
import { getMediaUrl } from '../../utils/api';
import type { Photo, MedicalData, ExtractedField } from '../../types/photo';

interface PhotoCardProps {
  photoId: string;
  imageUrl: string;
  timestamp: string;
  altText?: string;
  extractedText?: string;
  onDelete?: (photoId: string) => void;
  onUpdate?: (photoId: string, updatedData: Partial<Photo>) => Promise<void>;
  medicalData?: MedicalData;
}

const PhotoCard: React.FC<PhotoCardProps> = ({
  photoId,
  imageUrl,
  timestamp,
  altText = 'Photo',
  extractedText = '',
  onDelete,
  onUpdate,
  medicalData
}) => {
  const [isZoomed, setIsZoomed] = useState(false);
  const [imageError, setImageError] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [editingField, setEditingField] = useState<keyof MedicalData | null>(null);
  const [editData, setEditData] = useState<MedicalData | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const isEditing = editingField !== null;

  const handleImageError = () => {
    setImageError(true);
  };

  const toggleZoom = () => {
    setIsZoomed(!isZoomed);
  };

  const handleModalClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      setIsZoomed(false);
    }
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowDeleteConfirm(true);
  };

  const handleConfirmDelete = async () => {
    setIsDeleting(true);
    if (onDelete) {
      await onDelete(photoId);
    }
    setShowDeleteConfirm(false);
    setIsDeleting(false);
  };

  const getConfidenceColor = (confidence: number) => {
    if (confidence >= 0.95) return 'text-green-600';
    if (confidence >= 0.80) return 'text-yellow-600';
    return 'text-red-600';
  };

  const getConfidenceBg = (confidence: number, isEdited?: boolean, isValidated?: boolean) => {
    if (isEdited) return 'bg-blue-50 border-blue-200';
    if (isValidated) return 'bg-emerald-50 border-emerald-200';
    if (confidence >= 0.95) return 'bg-green-50 border-green-200';
    if (confidence >= 0.80) return 'bg-yellow-50 border-yellow-200';
    return 'bg-red-50 border-red-200';
  };

  const handleEditClick = (key: keyof MedicalData) => {
    if (medicalData) {
      setEditData(JSON.parse(JSON.stringify(medicalData)));
      setEditingField(key);
    }
  };

  const handleCancelEdit = () => {
    setEditingField(null);
    setEditData(null);
  };

  const handleSaveEdit = async () => {
    if (onUpdate && editData) {
      setIsSaving(true);
      try {
        await onUpdate(photoId, { medical_data: editData } as any);
        setEditingField(null);
        setEditData(null);
      } catch (error) {
        console.error('Failed to save edit:', error);
      } finally {
        setIsSaving(false);
      }
    }
  };

  const handleValidateAll = async () => {
    if (!onUpdate || !medicalData) return;
    
    setIsSaving(true);
    try {
      const validatedData = JSON.parse(JSON.stringify(medicalData)) as MedicalData;
      (Object.keys(validatedData) as Array<keyof MedicalData>).forEach(key => {
        const field = validatedData[key];
        if (field && typeof field === 'object' && 'value' in field) {
          (field as ExtractedField<unknown>).is_validated = true;
        }
      });
      await onUpdate(photoId, { medical_data: validatedData } as any);
    } catch (error) {
      console.error('Failed to validate data:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleValidateField = async (key: keyof MedicalData) => {
    if (!onUpdate || !medicalData) return;
    
    setIsSaving(true);
    try {
      const validatedData = JSON.parse(JSON.stringify(medicalData)) as MedicalData;
      const field = validatedData[key];
      if (field) {
        (field as ExtractedField<unknown>).is_validated = true;
      }
      await onUpdate(photoId, { medical_data: validatedData } as any);
    } catch (error) {
      console.error('Failed to validate field:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleFieldChange = (key: keyof MedicalData, value: string | boolean) => {
    if (editData) {
      setEditData({
        ...editData,
        [key]: {
          ...(editData[key] as ExtractedField<unknown>),
          value: value,
          is_edited: true,
          is_validated: true
        }
      } as MedicalData);
    }
  };

  const renderField = (label: string, key: keyof MedicalData, isBool: boolean = false) => {
    const isFieldEditing = editingField === key;
    const field = isFieldEditing ? (editData ? editData[key] : null) : (medicalData ? medicalData[key] : null);
    if (!field) return null;

    const typedField = field as ExtractedField<unknown>;

    return (
      <div className={`p-2 mb-2 rounded border transition-all ${getConfidenceBg(typedField.confidence, typedField.is_edited, typedField.is_validated)}`}>
        <div className="flex justify-between items-center mb-1">
          <div className="flex items-center gap-1">
            <span className="text-[10px] font-bold text-gray-500 uppercase tracking-tighter">{label}</span>
            {typedField.is_edited && (
              <span className="text-[9px] bg-blue-100 text-blue-600 px-1 rounded font-bold">EDITED</span>
            )}
            {typedField.is_validated && !typedField.is_edited && (
              <span className="text-[9px] bg-emerald-100 text-emerald-600 px-1 rounded font-bold">OK</span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <span className={`text-[10px] font-bold ${getConfidenceColor(typedField.confidence)}`}>
              {(typedField.confidence * 100).toFixed(0)}%
            </span>
            {!isEditing && (
              <div className="flex items-center gap-1">
                {!typedField.is_validated && !typedField.is_edited && (
                  <button
                    onClick={() => handleValidateField(key)}
                    className="text-emerald-500 hover:text-emerald-700 transition-colors p-0.5 rounded-full hover:bg-emerald-50"
                    title="Validate this field"
                  >
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                      <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                    </svg>
                  </button>
                )}
                <button
                  onClick={() => handleEditClick(key)}
                  className="text-sky-500 hover:text-sky-700 transition-colors p-0.5 rounded-full hover:bg-sky-50"
                  title="Edit this field"
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </button>
              </div>
            )}
          </div>
        </div>
        {isFieldEditing ? (
          <div className="flex flex-col gap-2">
            {isBool ? (
              <input
                type="checkbox"
                checked={!!typedField.value}
                onChange={(e) => handleFieldChange(key, e.target.checked)}
                className="w-4 h-4 text-sky-600 rounded focus:ring-sky-500"
              />
            ) : (
              <input
                type="text"
                value={(typedField.value as string) || ''}
                onChange={(e) => handleFieldChange(key, e.target.value)}
                className="w-full px-2 py-1 text-xs border rounded focus:outline-none focus:ring-1 focus:ring-sky-500 bg-white"
                autoFocus
              />
            )}
            <div className="flex justify-end gap-1 mt-1">
              <button
                onClick={handleCancelEdit}
                className="text-[10px] px-2 py-0.5 bg-gray-100 hover:bg-gray-200 text-gray-600 rounded transition-colors font-bold"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveEdit}
                className="text-[10px] px-2 py-0.5 bg-sky-600 hover:bg-sky-700 text-white rounded transition-colors font-bold"
                disabled={isSaving}
              >
                {isSaving ? '...' : 'Save'}
              </button>
            </div>
          </div>
        ) : (
          <div className="text-sm font-medium text-gray-800 break-words leading-tight">
            {isBool ? (typedField.value ? 'Yes' : 'No') : ((typedField.value as string) || 'N/A')}
          </div>
        )}
      </div>
    );
  };

  const needsReview = medicalData ? Object.values(medicalData).some(field => {
    if (field && typeof field === 'object' && field !== null && 'confidence' in field) {
      const typedField = field as ExtractedField<unknown>;
      return typedField.confidence < 0.95 && !typedField.is_validated && !typedField.is_edited;
    }
    return false;
  }) : false;


  return (
    <>
      <div 
        className={`group bg-white rounded-lg shadow-md overflow-hidden transition-all hover:shadow-xl relative flex flex-col h-[280px] cursor-pointer ${needsReview ? 'ring-2 ring-orange-400 ring-inset' : ''}`}
        onClick={toggleZoom}
      >
        {/* Image only view for small card */}
        <div className="flex-1 overflow-hidden relative">
          <img
            src={imageError ? fallbackImage : getMediaUrl(imageUrl)}
            alt={altText}
            onError={handleImageError}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
          />
          
          {/* Status Badge */}
          {needsReview && (
            <div className="absolute top-2 left-2 z-10">
              <span className="bg-orange-500 text-white text-[10px] font-black px-2 py-1 rounded shadow-lg flex items-center gap-1 uppercase tracking-wider">
                <svg xmlns="http://www.w3.org/2000/svg" className="h-3 w-3" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
                Needs Review
              </span>
            </div>
          )}
          
          {/* Overlay controls */}
          <div className="absolute inset-0 bg-black/0 group-hover:bg-black/10 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
             <div className="bg-white/90 backdrop-blur-sm p-2 rounded-full shadow-lg text-sky-600">
               <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                 <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0zM10 7v3m0 0v3m0-3h3m-3 0H7" />
               </svg>
             </div>
          </div>

          <div className="absolute top-2 right-2 flex gap-2">
            <button
              onClick={(e) => { e.stopPropagation(); handleDeleteClick(e); }}
              className="bg-red-500/80 hover:bg-red-600 text-white rounded-full p-2 shadow-lg transition-all opacity-0 group-hover:opacity-100"
              title="Delete photo"
            >
              <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>
        </div>

        {/* Footer info */}
        <div className="p-2 border-t border-gray-100 bg-white flex justify-between items-center shrink-0">
          <span className="text-[10px] text-gray-400 font-mono tracking-tighter">{photoId.slice(-8)}</span>
          <span className="text-[10px] text-gray-500 font-medium">{new Date(timestamp).toLocaleDateString()}</span>
        </div>

        {/* Delete confirmation dialog */}
        {showDeleteConfirm && (
          <div className="absolute inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-20" onClick={(e) => e.stopPropagation()}>
            <div className="bg-white rounded-lg p-4 m-4 shadow-xl">
              <p className="text-gray-800 text-sm font-medium mb-4">Delete this photo?</p>
              <div className="flex gap-2 justify-center">
                <button
                  onClick={() => setShowDeleteConfirm(false)}
                  className="px-3 py-1.5 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors text-xs font-semibold"
                  disabled={isDeleting}
                >
                  Cancel
                </button>
                <button
                  onClick={handleConfirmDelete}
                  className="px-3 py-1.5 bg-red-500 hover:bg-red-600 text-white rounded-md transition-colors text-xs font-semibold"
                  disabled={isDeleting}
                >
                  {isDeleting ? '...' : 'Delete'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Improved zoom modal overlay */}
      {isZoomed && (
        <div
          className="fixed inset-0 bg-black/80 backdrop-blur-md flex items-center justify-center z-50 transition-opacity duration-300 p-4"
          onClick={handleModalClick}
        >
          <div
            className="relative bg-white rounded-2xl shadow-2xl w-full max-w-7xl max-h-[95vh] overflow-hidden transform transition-all duration-300 flex flex-col"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div className="bg-gray-50/50 backdrop-blur-sm border-b border-gray-200 flex justify-between items-center p-4">
              <div className="flex flex-col">
                <div className="flex items-center gap-2">
                  <h2 className="text-gray-900 text-lg font-bold leading-tight">{altText}</h2>
                  {needsReview && (
                    <span className="bg-orange-100 text-orange-600 text-[10px] font-black px-1.5 py-0.5 rounded border border-orange-200 uppercase tracking-wider">
                      Needs Review
                    </span>
                  )}
                </div>
                <span className="text-xs text-gray-500">ID: {photoId}</span>
              </div>
              
              <div className="flex gap-3 items-center">
                <div className="flex gap-2">
                  <button
                    onClick={handleValidateAll}
                    className="px-4 py-2 bg-emerald-600 text-white text-sm font-bold rounded-lg hover:bg-emerald-700 transition-all flex items-center gap-2 shadow-md active:scale-95 disabled:opacity-50"
                    disabled={isSaving}
                  >
                    {isSaving ? (
                      <div className="animate-spin h-3 w-3 border-2 border-white border-t-transparent rounded-full" />
                    ) : (
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                      </svg>
                    )}
                    Validate All
                  </button>
                </div>
                <button
                  className="bg-gray-100 hover:bg-gray-200 text-gray-500 rounded-full p-2.5 transition-all active:rotate-90 duration-300"
                  onClick={toggleZoom}
                >
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>

            <div className="flex-1 overflow-hidden flex flex-col md:flex-row">
              {/* Image column */}
              <div className="flex-1 bg-gray-900/5 p-6 flex items-center justify-center overflow-auto custom-scrollbar">
                <img
                  src={imageError ? fallbackImage : getMediaUrl(imageUrl)}
                  alt={altText}
                  className="max-w-full max-h-full object-contain rounded-lg shadow-2xl"
                />
              </div>

              {/* Data column */}
              <div className="w-full md:w-[400px] lg:w-[450px] bg-white border-l border-gray-200 flex flex-col">
                <div className="flex-1 overflow-y-auto p-5 custom-scrollbar">
                  <h3 className="text-xs font-black text-gray-400 uppercase tracking-widest mb-6 flex items-center gap-2">
                    <svg xmlns="http://www.w3.org/2000/svg" className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
                    </svg>
                    Extracted Medical Fields
                  </h3>
                  
                  <div className="space-y-1 pb-10">
                    {renderField('Medical Unit', 'unitate_medicala')}
                    {renderField('Address Unit', 'adresa_unitate_medicala')}
                    {renderField('Phone Unit', 'telefon_unitate_medicala')}
                    {renderField('File Number', 'numar_fisa')}
                    {renderField('Employer', 'societate_unitate')}
                    {renderField('Last Name', 'nume')}
                    {renderField('First Name', 'prenume')}
                    {renderField('CNP', 'cnp')}
                    {renderField('Job Title', 'profesie_functie')}
                    {renderField('Workplace', 'loc_de_munca')}
                    {renderField('Control Type', 'tip_control')}
                    {renderField('Control: Hire', 'control_angajare', true)}
                    {renderField('Control: Periodic', 'control_periodic', true)}
                    {renderField('Control: Adaptation', 'control_adaptare', true)}
                    {renderField('Control: Resumption', 'control_reluare', true)}
                    {renderField('Control: Supervision', 'control_supraveghere', true)}
                    {renderField('Control: Other', 'control_alte', true)}
                    {renderField('Result', 'aviz_medical')}
                    {renderField('Result: Fit', 'aviz_apt', true)}
                    {renderField('Result: Conditioned Fit', 'aviz_apt_conditionat', true)}
                    {renderField('Result: Temporary Unfit', 'aviz_inapt_temporar', true)}
                    {renderField('Result: Unfit', 'aviz_inapt', true)}
                    {renderField('Recommendations', 'recomandari')}
                    {renderField('Exam Date', 'data')}
                    {renderField('Next Exam Date', 'data_urm_examinari')}
                  </div>

                  {extractedText && (
                    <div className="mt-6 pt-6 border-t border-gray-100">
                      <h3 className="text-xs font-black text-gray-400 uppercase tracking-widest mb-4">Raw OCR Output</h3>
                      <div className="text-[11px] font-mono text-gray-600 bg-gray-50 p-4 rounded-xl border border-gray-100 whitespace-pre-wrap leading-relaxed">
                        {extractedText}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </>
  );
};

export default PhotoCard;
