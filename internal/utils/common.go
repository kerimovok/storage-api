package utils

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
)

// Common utilities used across the storage-api

// GetFileExtension extracts and normalizes the file extension
func GetFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return strings.TrimPrefix(ext, ".")
}

// GetFileExtensionFromHeader extracts extension from multipart file header
func GetFileExtensionFromHeader(file *multipart.FileHeader) string {
	return GetFileExtension(file.Filename)
}

// MatchesMimeType checks if a MIME type matches a pattern
func MatchesMimeType(actual, pattern string) bool {
	// Exact match
	if actual == pattern {
		return true
	}

	// Wildcard match (e.g., "text/*" matches "text/plain")
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(actual, prefix+"/")
	}

	return false
}

// IsValidMimeType checks if a MIME type matches any of the expected patterns
func IsValidMimeType(actual string, expectedPatterns []string) bool {
	for _, pattern := range expectedPatterns {
		if MatchesMimeType(actual, pattern) {
			return true
		}
	}
	return false
}

// ParseSizeString converts human-readable size strings to bytes
func ParseSizeString(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)

	// Handle bytes
	if strings.HasSuffix(sizeStr, "B") && !strings.HasSuffix(sizeStr, "KB") && !strings.HasSuffix(sizeStr, "MB") && !strings.HasSuffix(sizeStr, "GB") && !strings.HasSuffix(sizeStr, "TB") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			return size, nil
		}
	}

	// Handle KB
	if strings.HasSuffix(sizeStr, "KB") {
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024), nil
		}
	}

	// Handle MB
	if strings.HasSuffix(sizeStr, "MB") {
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024), nil
		}
	}

	// Handle GB
	if strings.HasSuffix(sizeStr, "GB") {
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024 * 1024), nil
		}
	}

	// Handle TB
	if strings.HasSuffix(sizeStr, "TB") {
		sizeStr = strings.TrimSuffix(sizeStr, "TB")
		if size, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(size * 1024 * 1024 * 1024 * 1024), nil
		}
	}

	// Try to parse as raw bytes
	if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		return size, nil
	}

	return 0, fmt.Errorf("invalid size format: %s", sizeStr)
}
