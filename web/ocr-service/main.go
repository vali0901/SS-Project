package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	pb "ocr-service/ocr-service/proto"

	"github.com/otiai10/gosseract/v2"
	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// OCRServer implements the OCRService
type OCRServer struct {
	pb.UnimplementedOCRServiceServer
	ocrClient *gosseract.Client
}

// NewOCRServer creates and initializes a new OCRServer
func NewOCRServer() (*OCRServer, error) {
	client := gosseract.NewClient()
	// Configure language support: English and Romanian
	client.SetLanguage("eng", "ron")

	// Configure Tesseract for medical document OCR
	client.SetConfigFile("")
	client.SetVariable("tessedit_char_blacklist", "")

	return &OCRServer{
		ocrClient: client,
	}, nil
}

// ExtractText implements the OCR extraction RPC
func (s *OCRServer) ExtractText(ctx context.Context, req *pb.OCRRequest) (*pb.OCRResponse, error) {
	if len(req.ImageData) == 0 {
		return &pb.OCRResponse{
			Error: "image data is empty",
		}, nil
	}

	startTime := time.Now()

	// Set image from bytes
	s.ocrClient.SetImageFromBytes(req.ImageData)

	// Extract text - this is the critical operation that needs sandboxing
	text, err := s.ocrClient.Text()
	processingTime := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Printf("OCR extraction failed: %v", err)
		return &pb.OCRResponse{
			Error:            fmt.Sprintf("failed to extract text: %v", err),
			ProcessingTimeMs: processingTime,
		}, nil
	}

	log.Printf("Successfully extracted text from image in %dms (length: %d chars)", processingTime, len(text))

	return &pb.OCRResponse{
		Text:             text,
		ProcessingTimeMs: processingTime,
	}, nil
}

// Close closes the OCR client resources
func (s *OCRServer) Close() error {
	if s.ocrClient != nil {
		return s.ocrClient.Close()
	}
	return nil
}

func main() {
	// Create listener
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", port, err)
	}
	defer lis.Close()

	// Create OCR server
	ocrServer, err := NewOCRServer()
	if err != nil {
		log.Fatalf("failed to create OCR server: %v", err)
	}
	defer ocrServer.Close()

	// Create gRPC server
	s := grpc.NewServer()
	pb.RegisterOCRServiceServer(s, ocrServer)

	log.Printf("OCR service listening on %s", port)

	// Start serving
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
