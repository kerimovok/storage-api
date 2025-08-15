package constants

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"storage-api/internal/config"
	"storage-api/internal/utils"
)

// FileTypeInfo contains basic information about a file type (for backward compatibility)
type FileTypeInfo struct {
	Extensions   []string
	MimeTypes    []string
	MaxSizeBytes int64
	Description  string
	IsBlocked    bool
}

// ValidationResult contains the result of file validation
type ValidationResult struct {
	IsAllowed   bool
	MaxSize     int64
	RuleName    string
	Reason      string
	MatchedRule *config.ValidationRule
}

// ValidationEngine handles file validation using rules
type ValidationEngine struct {
	config config.FileValidationConfig
}

// NewValidationEngine creates a new validation engine
func NewValidationEngine(config config.FileValidationConfig) *ValidationEngine {
	return &ValidationEngine{
		config: config,
	}
}

// ValidateFile validates a file based on the configured rules
func (e *ValidationEngine) ValidateFile(filename, mimeType string, fileSize int64) *ValidationResult {
	// Get file extension
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" {
		ext = strings.TrimPrefix(ext, ".")
	}

	// Try to match rules
	for _, rule := range e.config.Rules {
		if e.matchesRule(ext, filename, mimeType, rule) {
			return e.applyRule(rule, fileSize, ext)
		}
	}

	// No rule matched, apply default action
	return e.applyDefaultAction(ext, filename, mimeType, fileSize)
}

// matchesRule checks if a file matches a validation rule
func (e *ValidationEngine) matchesRule(ext, filename, mimeType string, rule config.ValidationRule) bool {
	// Check extensions
	if len(rule.Extensions) > 0 {
		for _, allowedExt := range rule.Extensions {
			if strings.EqualFold(ext, allowedExt) {
				return true
			}
		}
	}

	// Check patterns (glob patterns like *.pdf)
	if len(rule.Patterns) > 0 {
		for _, pattern := range rule.Patterns {
			if e.matchesPattern(filename, pattern) {
				return true
			}
		}
	}

	// Check MIME types
	if len(rule.MimeTypes) > 0 {
		for _, allowedMime := range rule.MimeTypes {
			if e.matchesMimeType(mimeType, allowedMime) {
				return true
			}
		}
	}

	return false
}

// matchesPattern checks if a filename matches a glob pattern
func (e *ValidationEngine) matchesPattern(filename, pattern string) bool {
	// Convert glob pattern to regex
	regexPattern := e.globToRegex(pattern)
	matched, err := regexp.MatchString(regexPattern, filename)
	if err != nil {
		return false
	}
	return matched
}

// globToRegex converts a glob pattern to a regex pattern
func (e *ValidationEngine) globToRegex(pattern string) string {
	// Escape special regex characters
	pattern = regexp.QuoteMeta(pattern)

	// Convert glob wildcards to regex
	pattern = strings.ReplaceAll(pattern, "\\*", ".*")
	pattern = strings.ReplaceAll(pattern, "\\?", ".")

	// Add anchors
	return "^" + pattern + "$"
}

// matchesMimeType checks if a MIME type matches a pattern
func (e *ValidationEngine) matchesMimeType(actual, pattern string) bool {
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

// applyRule applies a validation rule to a file
func (e *ValidationEngine) applyRule(rule config.ValidationRule, fileSize int64, ext string) *ValidationResult {
	result := &ValidationResult{
		IsAllowed:   rule.Allow,
		RuleName:    rule.Name,
		MatchedRule: &rule,
	}

	if !rule.Allow {
		result.Reason = fmt.Sprintf("File blocked by rule '%s'", rule.Name)
		return result
	}

	// Check file size if rule has a size limit
	if rule.MaxSize != "" {
		maxSize, err := utils.ParseSizeString(rule.MaxSize)
		if err != nil {
			result.Reason = fmt.Sprintf("Invalid size limit in rule '%s': %s", rule.Name, rule.MaxSize)
			return result
		}

		if fileSize > maxSize {
			result.Reason = fmt.Sprintf("File size %s exceeds limit %s set by rule '%s'",
				FormatFileSize(fileSize), rule.MaxSize, rule.Name)
			return result
		}

		result.MaxSize = maxSize
	} else {
		// Use default size limit
		result.MaxSize = e.config.GetDefaultMaxFileSize()

		if fileSize > result.MaxSize {
			result.Reason = fmt.Sprintf("File size %s exceeds default limit %s",
				FormatFileSize(fileSize), FormatFileSize(result.MaxSize))
			return result
		}
	}

	return result
}

// applyDefaultAction applies the default action when no rules match
func (e *ValidationEngine) applyDefaultAction(ext, filename, mimeType string, fileSize int64) *ValidationResult {
	result := &ValidationResult{
		IsAllowed: !e.config.IsDefaultActionBlock(),
		RuleName:  "Default Action",
		MaxSize:   e.config.GetDefaultMaxFileSize(),
	}

	if e.config.IsDefaultActionBlock() {
		result.Reason = fmt.Sprintf("File type .%s not covered by any rules, default action is to block", ext)
	} else {
		result.Reason = fmt.Sprintf("File type .%s not covered by any rules, default action is to allow", ext)

		// Check against default size limit
		if fileSize > result.MaxSize {
			result.Reason = fmt.Sprintf("File size %s exceeds default limit %s",
				FormatFileSize(fileSize), FormatFileSize(result.MaxSize))
		}
	}

	return result
}

// GetAllowedExtensions returns all extensions that are allowed by rules
func (e *ValidationEngine) GetAllowedExtensions() []string {
	var allowed []string
	seen := make(map[string]bool)

	for _, rule := range e.config.Rules {
		if rule.Allow {
			for _, ext := range rule.Extensions {
				if !seen[ext] {
					allowed = append(allowed, ext)
					seen[ext] = true
				}
			}
		}
	}

	return allowed
}

// GetBlockedExtensions returns all extensions that are blocked by rules
func (e *ValidationEngine) GetBlockedExtensions() []string {
	var blocked []string
	seen := make(map[string]bool)

	for _, rule := range e.config.Rules {
		if !rule.Allow {
			for _, ext := range rule.Extensions {
				if !seen[ext] {
					blocked = append(blocked, ext)
					seen[ext] = true
				}
			}
		}
	}

	return blocked
}

// GetRuleByName finds a rule by its name
func (e *ValidationEngine) GetRuleByName(name string) *config.ValidationRule {
	for _, rule := range e.config.Rules {
		if rule.Name == name {
			return &rule
		}
	}
	return nil
}

// FormatFileSize formats bytes into human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
