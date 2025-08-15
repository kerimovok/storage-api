package config

import (
	"fmt"
	"log"
	"os"
	"storage-api/internal/utils"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kerimovok/go-pkg-utils/config"
	"gopkg.in/yaml.v3"
)

// ValidationRule represents a file validation rule
type ValidationRule struct {
	Name       string   `yaml:"name"`
	Extensions []string `yaml:"extensions,omitempty"`
	Patterns   []string `yaml:"patterns,omitempty"`
	MimeTypes  []string `yaml:"mime_types,omitempty"`
	MaxSize    string   `yaml:"max_size,omitempty"`
	Allow      bool     `yaml:"allow"`
}

// FileValidationConfig holds file validation settings
type FileValidationConfig struct {
	DefaultMaxSize       string           `yaml:"default_max_size"`
	DefaultAction        string           `yaml:"default_action"`
	StrictMimeValidation bool             `yaml:"strict_mime_validation"`
	Rules                []ValidationRule `yaml:"rules"`
}

// UploadConfig holds upload settings
type UploadConfig struct {
	MaxFiles     int    `yaml:"max_files"`
	MaxTotalSize string `yaml:"max_total_size"`
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
	Upload       UploadConfig              `yaml:"upload"`
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

// GetDefaultMaxFileSize returns the default max file size in bytes
func (c *FileValidationConfig) GetDefaultMaxFileSize() int64 {
	size, err := utils.ParseSizeString(c.DefaultMaxSize)
	if err != nil {
		log.Printf("Warning: Invalid default max file size '%s', using 10MB as fallback", c.DefaultMaxSize)
		return 10 * 1024 * 1024 // 10MB fallback
	}
	return size
}

// IsDefaultActionBlock returns true if the default action is to block files
func (c *FileValidationConfig) IsDefaultActionBlock() bool {
	return strings.ToLower(c.DefaultAction) == "block"
}

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
