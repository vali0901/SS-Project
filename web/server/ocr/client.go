package ocr

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "mqtt-streaming-server/mqtt-streaming-server/proto/ocr"

	"google.golang.org/grpc"
)

// Client wraps the gRPC connection and service stub
type Client struct {
	conn *grpc.ClientConn
	stub pb.OCRServiceClient
}

// NewClient creates a new OCR client connected to the OCR service
func NewClient(host string, port string) (*Client, error) {
	// Build address
	addr := fmt.Sprintf("%s:%s", host, port)

	// Connect to OCR service with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OCR service at %s: %w", addr, err)
	}

	stub := pb.NewOCRServiceClient(conn)

	log.Printf("Connected to OCR service at %s", addr)

	return &Client{
		conn: conn,
		stub: stub,
	}, nil
}

// ExtractText calls the remote OCR service to extract text from an image
func (c *Client) ExtractText(imageData []byte) (string, error) {
	if len(imageData) == 0 {
		return "", fmt.Errorf("image data is empty")
	}

	// Create request
	req := &pb.OCRRequest{
		ImageData: imageData,
	}

	// Call OCR service with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := c.stub.ExtractText(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OCR service error: %w", err)
	}

	// Check for service-level errors
	if resp.Error != "" {
		return "", fmt.Errorf("OCR extraction failed: %s", resp.Error)
	}

	log.Printf("OCR extraction completed in %dms", resp.ProcessingTimeMs)

	return resp.Text, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
