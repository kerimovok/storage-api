package requests

import (
	"time"
)

// UploadFileRequest represents a file upload request
type UploadFileRequest struct{}

// UpdateFileRequest represents a file update request
type UpdateFileRequest struct {
	FileName *string `json:"fileName,omitempty"`
	Status   *string `json:"status,omitempty" validate:"omitempty,oneof=active inactive archived deleted"`
}

// FileSearchRequest represents a file search request
type FileSearchRequest struct {
	Query          string     `json:"query,omitempty"`
	FileType       string     `json:"fileType,omitempty"`
	Status         string     `json:"status,omitempty" validate:"omitempty,oneof=active inactive archived deleted"`
	UploadedAfter  *time.Time `json:"uploadedAfter,omitempty"`
	UploadedBefore *time.Time `json:"uploadedBefore,omitempty"`
	Page           int        `json:"page" validate:"min=1"`
	Limit          int        `json:"limit" validate:"min=1,max=100"`
	SortBy         string     `json:"sortBy" validate:"omitempty,oneof=created_at updated_at original_name file_size"`
	SortOrder      string     `json:"sortOrder" validate:"omitempty,oneof=asc desc"`
}
