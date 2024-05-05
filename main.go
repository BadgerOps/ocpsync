package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type fileData struct {
	sha256sum string
	filename  string
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
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
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
			logrus.Error("Failed to download file", err)
		}
		fileList, err := generateFileList(config.OutputDir, version, config.IgnoredFiles)
		if err != nil {
			logrus.Error(err)
		}
		downloadFileList(fileList, url, version, config.OutputDir)
	}
}

func downloadFileList(fileList []fileData, url string, version string, outputDir string) {
	// try to download each file 3 times with exponential backoff on error
	const maxRetries = 3
	const initialBackoff = 1 * time.Second << maxRetries
	var err error
	for _, line := range fileList {
		for i := 0; i < maxRetries; i++ {
			err = validateFile(version, line.filename, line.sha256sum, outputDir)
			if err == nil {
				logrus.Infof("File validated! %s matches %s", line.sha256sum, line.filename)
				break
			}
			//logrus.Warnf("Could not validate local file %s, error: %s", url, err)
			err = downloadFile(url, outputDir, version, line.filename)
			if err != nil {
				logrus.Warnf("Failed to download %s, error: %s", url, err)
				time.Sleep(initialBackoff * (1 << uint(i)))
				continue
			}
			logrus.Debugf("Validating file %s at path %s", line.filename, outputDir)
			err = validateFile(version, line.filename, line.sha256sum, outputDir)
			if err != nil {
				logrus.Error("Failed to validate file: ", line.filename)
			}
		}
	}
	logrus.Info("Finished processing: ", version)
}

type list []string

func generateFileList(outputDir string, version string, ignoredFiles list) ([]fileData, error) {
	var file fileData
	fileSlice := make([]fileData, 0)
	fp := fmt.Sprintf("%s/%s/sha256sum.txt", outputDir, version)
	raw, err := ioutil.ReadFile(fp)
	if err != nil {
		logrus.Error("Could not open file path: ", err)
	}
	lines := strings.Split(string(raw), "\n")
	for _, line := range lines {
		if !containsAny(line, ignoredFiles) {
			if line != "" {
				fileInfo := strings.Split(line, " ")
				if len(fileInfo) < 3 {
					logrus.Warn("This file is not good?", fileInfo)
					break
				}
				file.sha256sum, file.filename = fileInfo[0], fileInfo[2]
				fileSlice = append(fileSlice, file)
			}
		}
	}
	return fileSlice, nil
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

func downloadFile(url string, outputDir string, filepath string, filename string) error {
	logrus.Debugf("Downloading file %s to path %s/%s from url %s ", filename, outputDir, filepath, url)
	fetchUrl := url + "/" + filename
	resp, err := http.Get(fetchUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fullPath := outputDir + "/" + filepath

	os.MkdirAll(fullPath, 0755)

	out, err := os.Create(fullPath + "/" + filename)
	if err != nil {
		logrus.Error("Could not create filepath ", fullPath)
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func validateFile(filepath, filename string, sha256sum string, outputDir string) error {
	fullPath := outputDir + "/" + filepath
	fileData, err := ioutil.ReadFile(fullPath + "/" + filename)
	if err != nil {
		return err
	}
	computedSum := sha256.Sum256(fileData)
	hexSum := hex.EncodeToString(computedSum[:])
	if hexSum != sha256sum {
		return fmt.Errorf("file validation failed: expected %s, got %s", sha256sum, hexSum)
	}
	return nil
}
