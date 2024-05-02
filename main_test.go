package main

import (
	"io/ioutil"
	"os"
	"strings"
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
	version := "/1.2.3/"
	filename := "testfile.txt"
	outputDir := tempDir

	// Call the downloadFile function
	err = downloadFile(url, outputDir, version, filename)
	if err != nil {
		t.Errorf("downloadFile returned an error: %v", err)
	}

	// Verify that the file was downloaded successfully
	_, err = os.Stat(outputDir + version + filename)
	if err != nil {
		t.Errorf("downloaded file does not exist: %v", err)
	}
}
func TestValidateFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary file for testing
	filepath := tempDir + "/1.2.3"
	os.MkdirAll(filepath, 0755)
	tempFile, err := ioutil.TempFile(filepath, "testfile")
	if err != nil {
		t.Fatal(err)
	}
	defer tempFile.Close()

	// Get the actual filename generated by ioutil.TempFile - its the last item if we split on /
	filename := strings.Split(tempFile.Name(), "/")[4]
	outputDir := tempDir + "/"

	// Write test data to the temporary file
	testData := []byte("test data")
	_, err = tempFile.Write(testData)
	if err != nil {
		t.Fatal(err)
	}

	// Define the filepath and expected sha256sum for testing (should always match this!)
	expectedSum := "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9"
	version := "1.2.3"

	// Call the validateFile function
	err = validateFile(version, filename, expectedSum, outputDir)
	if err != nil {
		t.Errorf("validateFile returned an error: %v", err)
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
