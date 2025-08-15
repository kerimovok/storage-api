package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/kerimovok/go-pkg-utils/config"
	"gopkg.in/yaml.v3"
)

// FileValidationConfig holds file validation settings
type FileValidationConfig struct {
	MaxFileSizePerExtension map[string]int64 `yaml:"max_file_size_per_extension"`
	DefaultMaxFileSize      int64            `yaml:"default_max_file_size"`
	AllowedExtensions       []string         `yaml:"allowed_extensions"`
	BlockedExtensions       []string         `yaml:"blocked_extensions"`
	StrictMimeValidation    bool             `yaml:"strict_mime_validation"`
}

// FileNamingConfig holds file naming strategy settings
type FileNamingConfig struct {
	Strategy          string `yaml:"strategy"`
	PreserveExtension bool   `yaml:"preserve_extension"`
}

// StorageOrganizationConfig holds file organization settings
type StorageOrganizationConfig struct {
	Pattern     string           `yaml:"pattern"`
	DateFormat  string           `yaml:"date_format"`
	IncludeTime bool             `yaml:"include_time"`
	Naming      FileNamingConfig `yaml:"naming"`
}

// LocalStorageConfig holds local storage settings
type LocalStorageConfig struct {
	UploadDir  string `yaml:"upload_dir"`
	CreateDirs bool   `yaml:"create_dirs"`
}

// StorageConfig holds the complete storage configuration
type StorageConfig struct {
	Validation   FileValidationConfig      `yaml:"validation"`
	Organization StorageOrganizationConfig `yaml:"organization"`
	Storage      LocalStorageConfig        `yaml:"storage"`
}

// MainConfig holds the root configuration
type MainConfig struct {
	Storage StorageConfig `yaml:"storage"`
}

var (
	Config MainConfig
)

// LoadConfig loads the configuration from the specified path
func LoadConfig() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		if config.GetEnv("GO_ENV") != "production" {
			log.Println("Warning: Failed to load .env file")
		}
	}

	// Read config file
	data, err := os.ReadFile("config/storage.yaml")
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config MainConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Store config globally
	Config = config

	log.Println("Storage configuration loaded successfully from config/storage.yaml")
	return nil
}

// GetConfig returns the current configuration
func GetConfig() MainConfig {
	return Config
}
