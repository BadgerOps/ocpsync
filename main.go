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

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	BaseURL   string   `yaml:"baseURL"`
	Version   []string `yaml:"version"`
	OutputDir string   `yaml:"outputDir"`
}

func init() {
	formatter := &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logrus.SetFormatter(formatter)
	logrus.SetLevel(logrus.DebugLevel)
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
	for _, version := range config.Version {
		logrus.Info("Processing files for version: ", version)
		url := config.BaseURL + version
		resp, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		paths := strings.Split(string(body), "\n")
		for _, path := range paths {
			if strings.Contains(path, version) {
				shaURL := url + "/sha256sum.txt"
				err = downloadFile(shaURL, version, "sha256sum.txt")
				if err != nil {
					logrus.Error("Failed to download file", err)
				}
				fileList, err := generateFileList(version)
				if err != nil {
					logrus.Error(err)
				}
				downloadFileList(fileList, url, err, version)
			}
		}
	}
}

func downloadFileList(fileList []byte, url string, err error, version string) {
	files := strings.Split(string(fileList), "\n")
	for _, file := range files {

		fileInfo := strings.Split(file, " ")
		if len(fileInfo) < 3 {
			logrus.Warn("This file is not good?", fileInfo)
			break
		}
		sha256sum := fileInfo[0]
		filename := fileInfo[2]
		fileURL := url + "/" + filename
		logrus.Debugln("Downloading: ", fileURL)
		err = validateFile(version, filename, sha256sum)
		if err != nil {
			err = downloadFile(fileURL, version, filename)
			if err != nil {
				panic(err)
			}
			err = validateFile(version, filename, sha256sum)
			if err != nil {
				logrus.Error("Failed to download file: ", fileURL)
				continue
			}

		}

	}
	logrus.Info("Finished processing: ", version)
}

func generateFileList(version string) ([]byte, error) {
	fp := fmt.Sprintf(version + "/sha256sum.txt")
	raw, err := ioutil.ReadFile(fp)
	if err != nil {
		panic(err)
	}

	lines := strings.Split(string(raw), "\n")
	filteredLines := []string{}

	for _, line := range lines {
		if !strings.Contains(line, "windows") && !strings.Contains(line, "arm64") && !strings.Contains(line, "aarch64") && !strings.Contains(line, "mac") && !strings.Contains(line, "pp64le") {
			if line != "" {
				filteredLines = append(filteredLines, line)
			}
		}
	}

	filteredRaw := []byte(strings.Join(filteredLines, "\n"))
	return filteredRaw, nil
}

func downloadFile(url string, filepath string, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	os.MkdirAll(filepath, 0755)

	out, err := os.Create(filepath + "/" + filename)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func validateFile(filepath, filename string, sha256sum string) error {
	fileData, err := ioutil.ReadFile(filepath + "/" + filename)
	if err != nil {
		return err
	}
	computedSum := sha256.Sum256(fileData)
	hexSum := hex.EncodeToString(computedSum[:])
	if hexSum != sha256sum {
		return fmt.Errorf("file validation failed: expected %s, got %s", sha256sum, hexSum)
	}
	logrus.Debugf("File validated! %s matches %s", filename, sha256sum)
	return nil
}
