package models

import (
	"github.com/kerimovok/go-pkg-database/sql"
)

// File represents a stored file
type File struct {
	sql.BaseModel
	OriginalName string `json:"originalName" gorm:"not null"`
	StoredName   string `json:"storedName" gorm:"not null;uniqueIndex"`
	FilePath     string `json:"filePath" gorm:"not null"`
	FileSize     int64  `json:"fileSize" gorm:"not null"`
	MimeType     string `json:"mimeType" gorm:"not null"`
	Extension    string `json:"extension" gorm:"not null"`
	FileType     string `json:"fileType" gorm:"not null"`
	Hash         string `json:"hash" gorm:"not null;uniqueIndex"`
	Status       string `json:"status" gorm:"not null;default:'active'"`
}
