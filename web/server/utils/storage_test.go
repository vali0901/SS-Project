package utils

import (
	"os"
	"testing"
)

func TestSaveAndGetLocalURL(t *testing.T) {
	file := "test_upload.txt"
	data := []byte("hello world")

	err := SaveToLocal(data, file)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	url := GetLocalURL(file)
	if url == "" {
		t.Fatal("expected url")
	}

	os.Remove(file)
}