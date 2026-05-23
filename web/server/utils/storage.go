package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func PhotoStorageKey(photoID string) string {
	return fmt.Sprintf("photos/%s.jpg", photoID)
}

func SaveToLocal(data []byte, keyName string) error {
	// keyName is like "photos/<uuid>.jpg".
	path := filepath.Join("uploads", keyName)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GetLocalURL(keyName string) string {
	// Construct URL assuming the server serves /uploads/ at root.
	baseURL := os.Getenv("API_BASE_URL")
	if baseURL == "" {
		return fmt.Sprintf("/uploads/%s", keyName)
	}

	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/uploads/%s", baseURL, keyName)
}
