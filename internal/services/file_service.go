package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"storage-api/internal/config"
	"storage-api/internal/constants"
	"storage-api/internal/utils"

	"github.com/google/uuid"
	"github.com/kerimovok/go-pkg-utils/errors"
)

// FileService handles all file operations and eliminates redundancy
type FileService struct {
	config           config.StorageConfig
	validationEngine *constants.ValidationEngine
}

// NewFileService creates a new file service instance
func NewFileService() *FileService {
	storageConfig := config.GetConfig().Storage
	return &FileService{
		config:           storageConfig,
		validationEngine: constants.NewValidationEngine(storageConfig.Validation),
	}
}

// ValidateFile validates the uploaded file
func (s *FileService) ValidateFile(file *multipart.FileHeader) error {
	// Get MIME type from file header
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Validate file using rules
	validationResult := s.validationEngine.ValidateFile(file.Filename, mimeType, file.Size)

	if !validationResult.IsAllowed {
		return errors.BadRequestError("FILE_BLOCKED", validationResult.Reason)
	}

	// MIME type validation if enabled
	if s.config.Validation.StrictMimeValidation {
		if err := s.validateMimeType(file, validationResult); err != nil {
			return err
		}
	}

	return nil
}

// ValidateMultipleFiles validates multiple uploaded files
func (s *FileService) ValidateMultipleFiles(files []*multipart.FileHeader) error {
	// Check maximum number of files
	if len(files) > s.config.Upload.MaxFiles {
		return errors.BadRequestError("TOO_MANY_FILES", fmt.Sprintf("Maximum %d files allowed per upload", s.config.Upload.MaxFiles))
	}

	// Calculate total size
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	// Check total size limit
	maxTotalSize, err := utils.ParseSizeString(s.config.Upload.MaxTotalSize)
	if err != nil {
		// If parsing fails, use a reasonable default
		maxTotalSize = 100 * 1024 * 1024 // 100MB
	}

	if totalSize > maxTotalSize {
		return errors.BadRequestError("TOTAL_SIZE_EXCEEDED", fmt.Sprintf("Total file size %s exceeds limit %s",
			constants.FormatFileSize(totalSize), s.config.Upload.MaxTotalSize))
	}

	// Validate each individual file
	for _, file := range files {
		if err := s.ValidateFile(file); err != nil {
			return err
		}
	}

	return nil
}

// validateMimeType validates the MIME type of the file
func (s *FileService) validateMimeType(file *multipart.FileHeader, validationResult *constants.ValidationResult) error {
	// Open file to check MIME type
	src, err := file.Open()
	if err != nil {
		return errors.InternalError("FILE_OPEN_ERROR", "Failed to open file for MIME type validation")
	}
	defer src.Close()

	// Read first 512 bytes for MIME type detection
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return errors.InternalError("FILE_READ_ERROR", "Failed to read file for MIME type validation")
	}

	// Detect MIME type
	detectedType := http.DetectContentType(buffer)

	// If we have a matched rule with MIME types, validate against them
	if validationResult.MatchedRule != nil && len(validationResult.MatchedRule.MimeTypes) > 0 {
		if err := s.validateMimeTypeAgainstRule(detectedType, validationResult.MatchedRule); err != nil {
			return err
		}
	}

	return nil
}

// validateMimeTypeAgainstRule validates MIME type against rule requirements
func (s *FileService) validateMimeTypeAgainstRule(detectedType string, rule *config.ValidationRule) error {
	valid := utils.IsValidMimeType(detectedType, rule.MimeTypes)

	if !valid {
		expectedTypes := strings.Join(rule.MimeTypes, ", ")
		return errors.BadRequestError("MIME_TYPE_MISMATCH", fmt.Sprintf("Expected MIME type for files matching rule '%s', got %s. Expected types: %s",
			rule.Name, detectedType, expectedTypes))
	}

	return nil
}

// GenerateFilePath generates the file path based on organization pattern
func (s *FileService) GenerateFilePath(originalName, fileType string) (string, string, error) {
	var pathParts []string

	// Add date component
	if strings.Contains(s.config.Organization.Pattern, "date") {
		dateFormat := s.config.Organization.DateFormat
		if s.config.Organization.IncludeTime {
			dateFormat = "2006-01-02-15"
		}
		pathParts = append(pathParts, time.Now().Format(dateFormat))
	}

	// Add file type component
	if strings.Contains(s.config.Organization.Pattern, "type") {
		pathParts = append(pathParts, fileType)
	}

	// Generate file name
	fileName, err := s.generateFileName(originalName)
	if err != nil {
		return "", "", err
	}

	// Combine path
	filePath := filepath.Join(pathParts...)
	fullPath := filepath.Join(s.config.Storage.UploadDir, filePath, fileName)

	return fullPath, fileName, nil
}

// generateFileName generates a unique file name
func (s *FileService) generateFileName(originalName string) (string, error) {
	ext := filepath.Ext(originalName)

	switch s.config.Organization.Naming.Strategy {
	case "uuid":
		id, err := uuid.NewRandom()
		if err != nil {
			return "", errors.InternalError("UUID_GENERATION_ERROR", "Failed to generate UUID")
		}
		if s.config.Organization.Naming.PreserveExtension {
			return id.String() + ext, nil
		}
		return id.String(), nil

	case "timestamp":
		timestamp := time.Now().UnixNano()
		if s.config.Organization.Naming.PreserveExtension {
			return fmt.Sprintf("%d%s", timestamp, ext), nil
		}
		return fmt.Sprintf("%d", timestamp), nil

	case "original":
		if s.config.Organization.Naming.PreserveExtension {
			return originalName, nil
		}
		return strings.TrimSuffix(originalName, ext), nil

	default:
		return "", errors.InternalError("INVALID_NAMING_STRATEGY", "Invalid file naming strategy")
	}
}

// SaveFile saves the uploaded file to storage
func (s *FileService) SaveFile(file *multipart.FileHeader, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if s.config.Storage.CreateDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.InternalError("DIR_CREATION_ERROR", fmt.Sprintf("Failed to create directory: %v", err))
		}
	}

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return errors.InternalError("FILE_CREATION_ERROR", fmt.Sprintf("Failed to create destination file: %v", err))
	}
	defer dst.Close()

	// Open source file
	src, err := file.Open()
	if err != nil {
		return errors.InternalError("FILE_OPEN_ERROR", fmt.Sprintf("Failed to open source file: %v", err))
	}
	defer src.Close()

	// Copy file content
	if _, err = io.Copy(dst, src); err != nil {
		return errors.InternalError("FILE_COPY_ERROR", fmt.Sprintf("Failed to copy file content: %v", err))
	}

	return nil
}

// ProcessMultipleFiles processes multiple uploaded files
func (s *FileService) ProcessMultipleFiles(files []*multipart.FileHeader) ([]*FileUploadResult, error) {
	var results []*FileUploadResult

	for _, file := range files {
		// Determine file type from extension
		ext := utils.GetFileExtensionFromHeader(file)
		fileType := ext

		// Generate file path and name
		filePath, storedName, err := s.GenerateFilePath(file.Filename, fileType)
		if err != nil {
			results = append(results, &FileUploadResult{
				OriginalName: file.Filename,
				Success:      false,
				Error:        err.Error(),
			})
			continue
		}

		// Save file to storage
		if err := s.SaveFile(file, filePath); err != nil {
			results = append(results, &FileUploadResult{
				OriginalName: file.Filename,
				Success:      false,
				Error:        err.Error(),
			})
			continue
		}

		// Calculate file hash
		hash, err := s.CalculateFileHash(filePath)
		if err != nil {
			results = append(results, &FileUploadResult{
				OriginalName: file.Filename,
				Success:      false,
				Error:        err.Error(),
			})
			continue
		}

		// Add successful result
		results = append(results, &FileUploadResult{
			OriginalName: file.Filename,
			StoredName:   storedName,
			FilePath:     filePath,
			FileSize:     file.Size,
			MimeType:     file.Header.Get("Content-Type"),
			Extension:    ext,
			FileType:     fileType,
			Hash:         hash,
			Success:      true,
		})
	}

	return results, nil
}

// FileUploadResult contains the result of processing a single file
type FileUploadResult struct {
	OriginalName string `json:"original_name"`
	StoredName   string `json:"stored_name,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
	Extension    string `json:"extension,omitempty"`
	FileType     string `json:"file_type,omitempty"`
	Hash         string `json:"hash,omitempty"`
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
}

// CalculateFileHash calculates MD5 hash of the file
func (s *FileService) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.InternalError("FILE_OPEN_ERROR", "Failed to open file for hash calculation")
	}
	defer file.Close()

	// For now, return a placeholder hash
	// TODO: Implement actual hash calculation when needed
	return "placeholder_hash", nil
}

// GetMaxFileSizeForExtension returns the maximum allowed file size for a specific extension
func (s *FileService) GetMaxFileSizeForExtension(extension string) int64 {
	// Use validation engine to get max size
	validationResult := s.validationEngine.ValidateFile(extension, "", 0)
	return validationResult.MaxSize
}

// GetFileTypeInfo returns comprehensive information about a file type
func (s *FileService) GetFileTypeInfo(extension string) (*constants.FileTypeInfo, bool) {
	// This is now handled by the validation engine
	// For backward compatibility, we'll create a basic info structure
	validationResult := s.validationEngine.ValidateFile(extension, "", 0)

	if validationResult.MatchedRule != nil {
		info := &constants.FileTypeInfo{
			Extensions:   validationResult.MatchedRule.Extensions,
			MimeTypes:    validationResult.MatchedRule.MimeTypes,
			MaxSizeBytes: validationResult.MaxSize,
			Description:  validationResult.MatchedRule.Name,
			IsBlocked:    !validationResult.IsAllowed,
		}
		return info, true
	}

	return nil, false
}

// IsExtensionAllowed checks if a file extension is allowed
func (s *FileService) IsExtensionAllowed(extension string) bool {
	validationResult := s.validationEngine.ValidateFile(extension, "", 0)
	return validationResult.IsAllowed
}

// GetAllowedExtensions returns all allowed file extensions
func (s *FileService) GetAllowedExtensions() []string {
	return s.validationEngine.GetAllowedExtensions()
}

// GetBlockedExtensions returns all blocked file extensions
func (s *FileService) GetBlockedExtensions() []string {
	return s.validationEngine.GetBlockedExtensions()
}

// GetValidationRules returns all validation rules
func (s *FileService) GetValidationRules() []config.ValidationRule {
	return s.config.Validation.Rules
}

// GetValidationRuleByName returns a specific validation rule by name
func (s *FileService) GetValidationRuleByName(name string) *config.ValidationRule {
	return s.validationEngine.GetRuleByName(name)
}

// ValidateFileType validates a file type without uploading
func (s *FileService) ValidateFileType(extension string, size int64) *constants.ValidationResult {
	return s.validationEngine.ValidateFile(extension, "", size)
}

// GetValidationConfig returns the validation configuration
func (s *FileService) GetValidationConfig() config.FileValidationConfig {
	return s.config.Validation
}

// GetUploadConfig returns the upload configuration
func (s *FileService) GetUploadConfig() config.UploadConfig {
	return s.config.Upload
}

// GetFileInfo returns comprehensive information about a file
func (s *FileService) GetFileInfo(file *multipart.FileHeader) *FileInfo {
	ext := utils.GetFileExtensionFromHeader(file)
	validationResult := s.validationEngine.ValidateFile(ext, "", file.Size)

	info := &FileInfo{
		OriginalName:     file.Filename,
		Extension:        ext,
		Size:             file.Size,
		SizeFormatted:    constants.FormatFileSize(file.Size),
		Category:         validationResult.RuleName,
		Description:      validationResult.Reason,
		IsAllowed:        validationResult.IsAllowed,
		IsBlocked:        !validationResult.IsAllowed,
		MaxSize:          validationResult.MaxSize,
		MaxSizeFormatted: constants.FormatFileSize(validationResult.MaxSize),
		MimeTypes:        []string{}, // Will be populated if rule has MIME types
	}

	// Add MIME types if available
	if validationResult.MatchedRule != nil {
		info.MimeTypes = validationResult.MatchedRule.MimeTypes
	}

	return info
}

// FileInfo contains comprehensive information about a file
type FileInfo struct {
	OriginalName     string   `json:"original_name"`
	Extension        string   `json:"extension"`
	Size             int64    `json:"size"`
	SizeFormatted    string   `json:"size_formatted"`
	Category         string   `json:"category"`
	Description      string   `json:"description"`
	IsAllowed        bool     `json:"is_allowed"`
	IsBlocked        bool     `json:"is_blocked"`
	MaxSize          int64    `json:"max_size"`
	MaxSizeFormatted string   `json:"max_size_formatted"`
	MimeTypes        []string `json:"mime_types"`
}

// GetFileTypeStats returns statistics about supported file types
func (s *FileService) GetFileTypeStats() *FileTypeStats {
	allowedExtensions := s.validationEngine.GetAllowedExtensions()
	blockedExtensions := s.validationEngine.GetBlockedExtensions()

	stats := &FileTypeStats{
		TotalTypes:      len(allowedExtensions) + len(blockedExtensions),
		AllowedTypes:    len(allowedExtensions),
		BlockedTypes:    len(blockedExtensions),
		Categories:      make(map[string]int),
		CategoryDetails: make(map[string]CategoryDetail),
	}

	// Get rules from config
	for _, rule := range s.config.Validation.Rules {
		category := rule.Name
		stats.Categories[category] = len(rule.Extensions)

		stats.CategoryDetails[category] = CategoryDetail{
			Name:       category,
			Count:      len(rule.Extensions),
			Extensions: rule.Extensions,
			TotalSize:  0, // Will be calculated if needed
			MaxSize:    0, // Will be calculated if needed
		}
	}

	return stats
}

// FileTypeStats contains statistics about file types
type FileTypeStats struct {
	TotalTypes      int                       `json:"total_types"`
	AllowedTypes    int                       `json:"allowed_types"`
	BlockedTypes    int                       `json:"blocked_types"`
	Categories      map[string]int            `json:"categories"`
	CategoryDetails map[string]CategoryDetail `json:"category_details"`
}

// CategoryDetail contains detailed information about a category
type CategoryDetail struct {
	Name       string   `json:"name"`
	Count      int      `json:"count"`
	Extensions []string `json:"extensions"`
	TotalSize  int64    `json:"total_size"`
	MaxSize    int64    `json:"max_size"`
}
