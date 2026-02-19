package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	// Set up a test HTTP server
	expectedContent := "hello from test server"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, expectedContent)
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	version := "1.2.3"
	filename := "testfile.txt"

	err := downloadFile(ts.URL, tempDir, version, filename)
	if err != nil {
		t.Fatalf("downloadFile returned an error: %v", err)
	}

	// Verify the file was downloaded with correct content
	content, err := os.ReadFile(filepath.Join(tempDir, version, filename))
	if err != nil {
		t.Fatalf("could not read downloaded file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("downloaded content = %q, want %q", string(content), expectedContent)
	}
}

func TestDownloadFileHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	tempDir := t.TempDir()

	err := downloadFile(ts.URL, tempDir, "1.0.0", "missing.txt")
	if err == nil {
		t.Fatal("expected error for 404 response, got nil")
	}
}

func TestValidateFile(t *testing.T) {
	tempDir := t.TempDir()
	version := "test"
	filename := "testfile.txt"

	// Create the version subdirectory and write test data
	versionDir := filepath.Join(tempDir, version)
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testData := []byte("test data")
	err = os.WriteFile(filepath.Join(versionDir, filename), testData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Compute the expected sha256sum
	sum := sha256.Sum256(testData)
	expectedSum := hex.EncodeToString(sum[:])

	// Should pass with the correct checksum
	err = validateFile(version, filename, expectedSum, tempDir)
	if err != nil {
		t.Errorf("validateFile returned an error: %v", err)
	}

	// Should fail with a wrong checksum
	err = validateFile(version, filename, "0000000000000000000000000000000000000000000000000000000000000000", tempDir)
	if err == nil {
		t.Error("validateFile should have returned an error for wrong checksum")
	}
}

func TestValidateFileMissing(t *testing.T) {
	tempDir := t.TempDir()

	err := validateFile("noversion", "nofile.txt", "abc", tempDir)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestContainsAny(t *testing.T) {
	ignoredFiles := []string{"file1.txt", "file2.txt", "file3.txt"}

	// Test case 1: line contains an ignored file
	line1 := "This is file1.txt"
	if !containsAny(line1, ignoredFiles) {
		t.Errorf("containsAny returned false for line: %s", line1)
	}

	// Test case 2: line does not contain any ignored file
	line2 := "This is a test"
	if containsAny(line2, ignoredFiles) {
		t.Errorf("containsAny returned true for line: %s", line2)
	}

	// Test case 3: line contains multiple ignored files
	line3 := "This is file2.txt and file3.txt"
	if !containsAny(line3, ignoredFiles) {
		t.Errorf("containsAny returned false for line: %s", line3)
	}
}

func TestGenerateFileList(t *testing.T) {
	version := "1.2.3"
	ignoredFiles := []string{"file1.txt"}

	tempDir := t.TempDir()
	versionDir := filepath.Join(tempDir, version)
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testData := []byte("916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9 file1.txt\n" +
		"1234567890abcdef file2.txt\n" +
		"abcdef1234567890 file3.txt")
	err = os.WriteFile(filepath.Join(versionDir, "sha256sum.txt"), testData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	filteredRaw, err := generateFileList(tempDir, version, ignoredFiles)
	if err != nil {
		t.Fatalf("generateFileList returned an error: %v", err)
	}

	expectedFilteredRaw := []byte("1234567890abcdef file2.txt\nabcdef1234567890 file3.txt")
	if !bytes.Equal(filteredRaw, expectedFilteredRaw) {
		t.Errorf("generateFileList returned %q, want %q", filteredRaw, expectedFilteredRaw)
	}
}

func TestGenerateFileListMissingFile(t *testing.T) {
	tempDir := t.TempDir()

	_, err := generateFileList(tempDir, "nonexistent", []string{})
	if err == nil {
		t.Error("expected error for missing sha256sum.txt, got nil")
	}
}

func TestDownloadFileCreatesDirectory(t *testing.T) {
	expectedContent := "directory test"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, expectedContent)
	}))
	defer ts.Close()

	tempDir := t.TempDir()
	version := "nested/path"
	filename := "file.txt"

	err := downloadFile(ts.URL, tempDir, version, filename)
	if err != nil {
		t.Fatalf("downloadFile returned an error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(tempDir, version, filename))
	if err != nil {
		t.Fatalf("could not read downloaded file: %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("downloaded content = %q, want %q", string(content), expectedContent)
	}
}

func TestGenerateFileListFiltersEmptyLines(t *testing.T) {
	version := "1.0.0"
	tempDir := t.TempDir()
	versionDir := filepath.Join(tempDir, version)
	err := os.MkdirAll(versionDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Input with empty lines and trailing newline
	testData := []byte("abc123 file1.txt\n\ndef456 file2.txt\n")
	err = os.WriteFile(filepath.Join(versionDir, "sha256sum.txt"), testData, 0644)
	if err != nil {
		t.Fatal(err)
	}

	filteredRaw, err := generateFileList(tempDir, version, []string{})
	if err != nil {
		t.Fatalf("generateFileList returned an error: %v", err)
	}

	expected := []byte("abc123 file1.txt\ndef456 file2.txt")
	if !bytes.Equal(filteredRaw, expected) {
		t.Errorf("generateFileList returned %q, want %q", filteredRaw, expected)
	}
}
