package services

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"storage-api/internal/config"

	"github.com/google/uuid"
	"github.com/kerimovok/go-pkg-utils/errors"
)

// FileService handles file operations
type FileService struct {
	config config.StorageConfig
}

// NewFileService creates a new file service instance
func NewFileService() *FileService {
	return &FileService{
		config: config.GetConfig().Storage,
	}
}

// ValidateFile validates the uploaded file
func (s *FileService) ValidateFile(file *multipart.FileHeader) error {
	// Check file size
	if file.Size > s.config.Validation.MaxFileSize {
		return errors.BadRequestError("FILE_TOO_LARGE", fmt.Sprintf("File size exceeds maximum allowed size of %d bytes", s.config.Validation.MaxFileSize))
	}

	// Get file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		return errors.BadRequestError("INVALID_FILE", "File must have a valid extension")
	}
	ext = strings.TrimPrefix(ext, ".")

	// Check if extension is blocked
	for _, blocked := range s.config.Validation.BlockedExtensions {
		if ext == blocked {
			return errors.BadRequestError("BLOCKED_FILE_TYPE", fmt.Sprintf("File type .%s is not allowed", ext))
		}
	}

	// Check if extension is allowed
	allowed := false
	for _, allowedExt := range s.config.Validation.AllowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	if !allowed {
		return errors.BadRequestError("INVALID_FILE_TYPE", fmt.Sprintf("File type .%s is not allowed", ext))
	}

	// MIME type validation if enabled
	if s.config.Validation.StrictMimeValidation {
		if err := s.validateMimeType(file, ext); err != nil {
			return err
		}
	}

	return nil
}

// validateMimeType validates the MIME type of the file
func (s *FileService) validateMimeType(file *multipart.FileHeader, ext string) error {
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

	// Validate against expected MIME types for the extension
	if err := s.validateMimeTypeForExtension(detectedType, ext); err != nil {
		return err
	}

	return nil
}

// validateMimeTypeForExtension validates MIME type against file extension
func (s *FileService) validateMimeTypeForExtension(mimeType, ext string) error {
	// Define expected MIME types for common extensions
	expectedMimeTypes := map[string][]string{
		"jpg":  {"image/jpeg"},
		"jpeg": {"image/jpeg"},
		"png":  {"image/png"},
		"gif":  {"image/gif"},
		"pdf":  {"application/pdf"},
		"doc":  {"application/msword"},
		"docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		"xls":  {"application/vnd.ms-excel"},
		"xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		"txt":  {"text/plain"},
		"csv":  {"text/csv"},
		"zip":  {"application/zip"},
		"rar":  {"application/x-rar-compressed"},
		"mp4":  {"video/mp4"},
		"avi":  {"video/x-msvideo"},
		"mov":  {"video/quicktime"},
		"mp3":  {"audio/mpeg"},
		"wav":  {"audio/wav"},
	}

	if expectedTypes, exists := expectedMimeTypes[ext]; exists {
		valid := false
		for _, expectedType := range expectedTypes {
			if mimeType == expectedType {
				valid = true
				break
			}
		}
		if !valid {
			return errors.BadRequestError("MIME_TYPE_MISMATCH", fmt.Sprintf("Expected MIME type for .%s files, got %s", ext, mimeType))
		}
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

// CalculateFileHash calculates MD5 hash of the file
func (s *FileService) CalculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.InternalError("FILE_OPEN_ERROR", "Failed to open file for hash calculation")
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", errors.InternalError("HASH_CALCULATION_ERROR", "Failed to calculate file hash")
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// GenerateShareToken generates a secure share token
func (s *FileService) GenerateShareToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", errors.InternalError("TOKEN_GENERATION_ERROR", "Failed to generate share token")
	}
	return hex.EncodeToString(token), nil
}
