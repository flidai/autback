package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type Service struct {
	Image        string `json:"image,omitempty"`
	CACertFile   string `json:"ca_cert_file,omitempty"`
	OIDCAudience string `json:"oidc_audience,omitempty"`
}

type Config struct {
	URL     string   `json:"url"`
	Service *Service `json:"service,omitempty"`
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	config, fileErr := read(path)
	if fileErr != nil && !errors.Is(fileErr, os.ErrNotExist) {
		return Config{}, fileErr
	}
	if value := os.Getenv("AUTBACK_URL"); value != "" {
		config.URL = value
	}
	if config.URL == "" {
		return Config{}, fmt.Errorf("url is required in %s or AUTBACK_URL", path)
	}
	if config.Service == nil {
		config.Service = &Service{}
	}
	if config.Service.OIDCAudience == "" {
		config.Service.OIDCAudience = config.URL
	}
	return config, nil
}

func Path() (string, error) {
	if value := os.Getenv("AUTBACK_CONFIG"); value != "" {
		return value, nil
	}
	root := os.Getenv("XDG_CONFIG_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".config")
	}
	return filepath.Join(root, "autback", "config.json"), nil
}

func read(path string) (Config, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Config{}, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		return Config{}, fmt.Errorf("%s must not be accessible by group or other users", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	return config, nil
}
