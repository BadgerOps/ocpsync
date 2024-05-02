package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestDownloadFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary file for testing
	tempFile, err := ioutil.TempFile(tempDir, "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer tempFile.Close()

	// Define the URL, filepath, and filename for testing
	url := "https://example.com/file.txt"
	filepath := tempDir
	filename := "testfile.txt"

	// Call the downloadFile function
	err = downloadFile(url, filepath, filename)
	if err != nil {
		t.Errorf("downloadFile returned an error: %v", err)
	}

	// Verify that the file was downloaded successfully
	_, err = os.Stat(filepath + "/" + filename)
	if err != nil {
		t.Errorf("downloaded file does not exist: %v", err)
	}
}
