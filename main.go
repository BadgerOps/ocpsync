package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	OcpBinaries Section `yaml:"ocpbinaries"`
	Rhcos       Section `yaml:"rhcos"`
}

type Section struct {
	BaseURL      string   `yaml:"baseURL"`
	Version      []string `yaml:"version"`
	IgnoredFiles []string `yaml:"ignoredFiles"`
	OutputDir    string   `yaml:"outputDir"`
}

func init() {
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logrus.SetFormatter(formatter)
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		logrus.Fatal("Failed to read config file: ", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		logrus.Fatal("Failed to parse config file: ", err)
	}
	logrus.Info("Running downloads for RHCOS images")
	downloadHandler(config.Rhcos)
	logrus.Info("Running downloads for OCP Binaries")
	downloadHandler(config.OcpBinaries)
}

func downloadHandler(config Section) {
	for _, version := range config.Version {
		logrus.Info("Processing files for version: ", version)
		url := config.BaseURL + version
		err := downloadFile(url, config.OutputDir, version, "sha256sum.txt")
		if err != nil {
			logrus.Errorf("Failed to download sha256sum.txt for version %s: %s", version, err)
			continue
		}
		fileList, err := generateFileList(config.OutputDir, version, config.IgnoredFiles)
		if err != nil {
			logrus.Errorf("Failed to generate file list for version %s: %s", version, err)
			continue
		}
		downloadFileList(fileList, url, version, config.OutputDir)
	}
}

func downloadFileList(fileList []byte, url string, version string, outputDir string) {
	// given a list of files, download them line by line and validate them with the sha256sum
	files := strings.Split(string(fileList), "\n")

	for _, file := range files {
		fileInfo := strings.Split(file, " ")
		if len(fileInfo) < 3 {
			logrus.Warnf("Skipping malformed line: %v", fileInfo)
			continue
		}
		// split the 'fileInfo' line - it will have 3 items, a sha256sum, a space and the filename
		sha256sum := fileInfo[0]
		filename := fileInfo[2]

		// try to download each file 3 times with exponential backoff on error
		const maxRetries = 3
		const initialBackoff = 1 * time.Second
		var err error
		for i := 0; i < maxRetries; i++ {
			err = validateFile(version, filename, sha256sum, outputDir)
			if err == nil {
				logrus.Infof("File validated! %s matches %s", sha256sum, filename)
				break
			}
			err = downloadFile(url, outputDir, version, filename)
			if err != nil {
				logrus.Warnf("Failed to download %s/%s, error: %s", url, filename, err)
				time.Sleep(initialBackoff * (1 << uint(i)))
				continue
			}
			logrus.Debugf("Validating file %s at path %s", filename, outputDir)
			err = validateFile(version, filename, sha256sum, outputDir)
			if err != nil {
				logrus.Error("Failed to validate file: ", filename)
			}
		}
	}
	logrus.Info("Finished processing: ", version)
}

func generateFileList(outputDir string, version string, ignoredFiles []string) ([]byte, error) {
	fp := filepath.Join(outputDir, version, "sha256sum.txt")
	file, err := os.Open(fp)
	if err != nil {
		return nil, fmt.Errorf("could not open file path %s: %w", fp, err)
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file %s: %w", fp, err)
	}
	lines := strings.Split(string(raw), "\n")
	filteredLines := []string{}
	for _, line := range lines {
		if !containsAny(line, ignoredFiles) {
			if line != "" {
				filteredLines = append(filteredLines, line)
			}
		}
	}

	filteredRaw := []byte(strings.Join(filteredLines, "\n"))
	return filteredRaw, nil
}

func containsAny(line string, ignoredFiles []string) bool {
	for _, ignoredFile := range ignoredFiles {
		if strings.Contains(line, ignoredFile) {
			logrus.Debugf("Ignoring %s as it matches %s", ignoredFile, line)
			return true
		}
	}
	return false
}

func downloadFile(url string, outputDir string, version string, filename string) error {
	logrus.Debugf("Downloading file %s to path %s/%s from url %s ", filename, outputDir, version, url)
	fetchURL := url + "/" + filename
	resp, err := http.Get(fetchURL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, fetchURL)
	}

	fullPath := filepath.Join(outputDir, version)

	err = os.MkdirAll(fullPath, 0755)
	if err != nil {
		return fmt.Errorf("could not create directory %s: %w", fullPath, err)
	}
	out, err := os.Create(filepath.Join(fullPath, filename))
	if err != nil {
		return fmt.Errorf("could not create file %s: %w", filepath.Join(fullPath, filename), err)
	}

	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func validateFile(version, filename string, sha256sum string, outputDir string) error {
	fullPath := filepath.Join(outputDir, version, filename)
	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return err
	}
	hexSum := hex.EncodeToString(hasher.Sum(nil))
	if hexSum != sha256sum {
		return fmt.Errorf("file validation failed: expected %s, got %s", sha256sum, hexSum)
	}
	return nil
}
