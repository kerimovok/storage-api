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
	MaxFileSize          int64    `yaml:"max_file_size"`
	AllowedExtensions    []string `yaml:"allowed_extensions"`
	BlockedExtensions    []string `yaml:"blocked_extensions"`
	StrictMimeValidation bool     `yaml:"strict_mime_validation"`
}

// FileNamingConfig holds file naming strategy settings
type FileNamingConfig struct {
	Strategy          string `yaml:"strategy"`
	CustomFunction    string `yaml:"custom_function"`
	PreserveExtension bool   `yaml:"preserve_extension"`
}

// StorageOrganizationConfig holds file organization settings
type StorageOrganizationConfig struct {
	Pattern       string           `yaml:"pattern"`
	CustomPattern string           `yaml:"custom_pattern"`
	DateFormat    string           `yaml:"date_format"`
	IncludeTime   bool             `yaml:"include_time"`
	Naming        FileNamingConfig `yaml:"naming"`
}

// LocalStorageConfig holds local storage settings
type LocalStorageConfig struct {
	UploadDir       string `yaml:"upload_dir"`
	CreateDirs      bool   `yaml:"create_dirs"`
	FilePermissions string `yaml:"file_permissions"`
	DirPermissions  string `yaml:"dir_permissions"`
}

// SecurityConfig holds security settings
type SecurityConfig struct {
	SecureURLs       bool     `yaml:"secure_urls"`
	URLExpiration    int      `yaml:"url_expiration"`
	RequireAuth      bool     `yaml:"require_auth"`
	AllowedReferrers []string `yaml:"allowed_referrers"`
}

// CleanupConfig holds cleanup settings
type CleanupConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Interval      int    `yaml:"interval"`
	RetentionDays int    `yaml:"retention_days"`
	Strategy      string `yaml:"strategy"`
}

// StorageConfig holds the complete storage configuration
type StorageConfig struct {
	Validation   FileValidationConfig      `yaml:"validation"`
	Organization StorageOrganizationConfig `yaml:"organization"`
	Storage      LocalStorageConfig        `yaml:"storage"`
	Security     SecurityConfig            `yaml:"security"`
	Cleanup      CleanupConfig             `yaml:"cleanup"`
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
