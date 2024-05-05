package main

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
