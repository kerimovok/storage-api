package requests

import (
	"time"

	"github.com/google/uuid"
)

// UploadFileRequest represents a file upload request
type UploadFileRequest struct {
	FileName     string                 `json:"fileName" validate:"required"`
	FileType     string                 `json:"fileType" validate:"required"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt    *time.Time             `json:"expiresAt,omitempty"`
	UploadedBy   *uuid.UUID             `json:"uploadedBy,omitempty"`
	IsPublic     bool                   `json:"isPublic"`
	Password     string                 `json:"password,omitempty"`
	MaxDownloads *int                   `json:"maxDownloads,omitempty"`
}

// UpdateFileRequest represents a file update request
type UpdateFileRequest struct {
	FileName  *string                `json:"fileName,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt *time.Time             `json:"expiresAt,omitempty"`
	Status    *string                `json:"status,omitempty"`
	IsPublic  *bool                  `json:"isPublic,omitempty"`
}

// FileSearchRequest represents a file search request
type FileSearchRequest struct {
	Query          string     `json:"query,omitempty"`
	FileType       string     `json:"fileType,omitempty"`
	Status         string     `json:"status,omitempty"`
	UploadedBy     *uuid.UUID `json:"uploadedBy,omitempty"`
	UploadedAfter  *time.Time `json:"uploadedAfter,omitempty"`
	UploadedBefore *time.Time `json:"uploadedBefore,omitempty"`
	Page           int        `json:"page" validate:"min=1"`
	Limit          int        `json:"limit" validate:"min=1,max=100"`
	SortBy         string     `json:"sortBy" validate:"oneof=created_at updated_at file_name file_size"`
	SortOrder      string     `json:"sortOrder" validate:"oneof=asc desc"`
}

// ShareFileRequest represents a file sharing request
type ShareFileRequest struct {
	FileID       uuid.UUID  `json:"fileId" validate:"required"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads *int       `json:"maxDownloads,omitempty"`
	Password     string     `json:"password,omitempty"`
	IsPublic     bool       `json:"isPublic"`
}

// DownloadFileRequest represents a file download request
type DownloadFileRequest struct {
	FileID     uuid.UUID  `json:"fileId" validate:"required"`
	ShareToken string     `json:"shareToken,omitempty"`
	Password   string     `json:"password,omitempty"`
	UserID     *uuid.UUID `json:"userId,omitempty"`
	SessionID  string     `json:"sessionId,omitempty"`
}
