package broker

//go:generate mockgen -destination=../mocks/ocr_client_mock.go -package=mocks . OCRClientInterface

// OCRClientInterface defines the contract for OCR services
type OCRClientInterface interface {
	// ExtractText extracts text from image data
	ExtractText(imageData []byte) (string, error)
	// Close closes the client connection
	Close() error
}
