package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/kerimovok/go-pkg-database/sql"
)

// File represents a stored file
type File struct {
	sql.BaseModel
	OriginalName string     `json:"originalName" gorm:"not null"`
	StoredName   string     `json:"storedName" gorm:"not null;uniqueIndex"`
	FilePath     string     `json:"filePath" gorm:"not null"`
	FileSize     int64      `json:"fileSize" gorm:"not null"`
	MimeType     string     `json:"mimeType" gorm:"not null"`
	Extension    string     `json:"extension" gorm:"not null"`
	FileType     string     `json:"fileType" gorm:"not null"`
	Hash         string     `json:"hash" gorm:"not null;uniqueIndex"`
	Status       string     `json:"status" gorm:"not null;default:'active'"`
	UploadedBy   *uuid.UUID `json:"uploadedBy,omitempty"`
	Metadata     sql.JSONB  `json:"metadata" gorm:"type:jsonb"`
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`
	AccessCount  int64      `json:"accessCount" gorm:"default:0"`
	LastAccessed *time.Time `json:"lastAccessed,omitempty"`
}

// FileAccess represents file access logs
type FileAccess struct {
	sql.BaseModel
	FileID     uuid.UUID  `json:"fileId" gorm:"not null"`
	IPAddress  string     `json:"ipAddress"`
	UserAgent  string     `json:"userAgent"`
	Referer    string     `json:"referer"`
	AccessType string     `json:"accessType" gorm:"not null"` // download, view, delete
	AccessTime time.Time  `json:"accessTime" gorm:"not null"`
	UserID     *uuid.UUID `json:"userId,omitempty"`
	SessionID  string     `json:"sessionId"`
}

// FileShare represents file sharing information
type FileShare struct {
	sql.BaseModel
	FileID        uuid.UUID  `json:"fileId" gorm:"not null"`
	ShareToken    string     `json:"shareToken" gorm:"not null;uniqueIndex"`
	ExpiresAt     *time.Time `json:"expiresAt,omitempty"`
	MaxDownloads  *int       `json:"maxDownloads,omitempty"`
	DownloadCount int        `json:"downloadCount" gorm:"default:0"`
	Password      string     `json:"password,omitempty"`
	IsPublic      bool       `json:"isPublic" gorm:"default:false"`
	CreatedBy     *uuid.UUID `json:"createdBy,omitempty"`
}
