package handlers

import (
	"log"
	"os"
	"path/filepath"
	"storage-api/internal/database"
	"storage-api/internal/models"
	"storage-api/internal/requests"
	"storage-api/internal/services"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/kerimovok/go-pkg-database/sql"
	"github.com/kerimovok/go-pkg-utils/httpx"
	"github.com/kerimovok/go-pkg-utils/validator"
	"gorm.io/gorm"
)

// FileHandler handles file-related HTTP requests
type FileHandler struct {
	fileService *services.FileService
}

// NewFileHandler creates a new file handler
func NewFileHandler() *FileHandler {
	return &FileHandler{
		fileService: services.NewFileService(),
	}
}

// UploadFile handles file upload requests
func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	// Parse multipart form
	file, err := c.FormFile("file")
	if err != nil {
		response := httpx.BadRequest("No file provided", err)
		return httpx.SendResponse(c, response)
	}

	// Parse additional form data
	var input requests.UploadFileRequest
	if err := c.BodyParser(&input); err != nil {
		response := httpx.BadRequest("Invalid request body", err)
		return httpx.SendResponse(c, response)
	}

	// Validate request
	validationErrors := validator.ValidateStruct(&input)
	if validationErrors.HasErrors() {
		httpxErrors := make([]httpx.ValidationError, len(validationErrors))
		for i, err := range validationErrors {
			httpxErrors[i] = httpx.ValidationError{
				Field:   err.Field,
				Message: err.Message,
			}
		}
		response := httpx.UnprocessableEntityWithValidation("Validation failed", httpxErrors)
		return httpx.SendValidationResponse(c, response)
	}

	// Validate file
	if err := h.fileService.ValidateFile(file); err != nil {
		response := httpx.BadRequest("File validation failed", err)
		return httpx.SendResponse(c, response)
	}

	// Determine file type from extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	fileType := strings.TrimPrefix(ext, ".")

	// Generate file path and name
	filePath, storedName, err := h.fileService.GenerateFilePath(file.Filename, fileType)
	if err != nil {
		response := httpx.InternalServerError("Failed to generate file path", err)
		return httpx.SendResponse(c, response)
	}

	// Save file to storage
	if err := h.fileService.SaveFile(file, filePath); err != nil {
		response := httpx.InternalServerError("Failed to save file", err)
		return httpx.SendResponse(c, response)
	}

	// Calculate file hash
	hash, err := h.fileService.CalculateFileHash(filePath)
	if err != nil {
		response := httpx.InternalServerError("Failed to calculate file hash", err)
		return httpx.SendResponse(c, response)
	}

	// Create file record
	fileRecord := models.File{
		OriginalName: file.Filename,
		StoredName:   storedName,
		FilePath:     filePath,
		FileSize:     file.Size,
		MimeType:     file.Header.Get("Content-Type"),
		Extension:    ext,
		FileType:     fileType,
		Hash:         hash,
		Status:       "active",
		UploadedBy:   input.UploadedBy,
		Metadata:     sql.JSONB(input.Metadata),
		ExpiresAt:    input.ExpiresAt,
	}

	// Use transaction to save file record
	err = sql.WithTransaction(database.DB, func(tx *gorm.DB) error {
		if err := tx.Create(&fileRecord).Error; err != nil {
			return err
		}

		// Create file share if requested
		if input.IsPublic || input.Password != "" || input.MaxDownloads != nil {
			shareToken, err := h.fileService.GenerateShareToken()
			if err != nil {
				return err
			}

			fileShare := models.FileShare{
				FileID:       fileRecord.ID,
				ShareToken:   shareToken,
				ExpiresAt:    input.ExpiresAt,
				MaxDownloads: input.MaxDownloads,
				Password:     input.Password,
				IsPublic:     input.IsPublic,
				CreatedBy:    input.UploadedBy,
			}

			if err := tx.Create(&fileShare).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Printf("Failed to save file record: %v", err)
		response := httpx.InternalServerError("Failed to process file upload", err)
		return httpx.SendResponse(c, response)
	}

	response := httpx.Created("File uploaded successfully", fileRecord)
	return httpx.SendResponse(c, response)
}

// GetFile retrieves file information
func (h *FileHandler) GetFile(c *fiber.Ctx) error {
	id := c.Params("id")
	fileID, err := uuid.Parse(id)
	if err != nil {
		response := httpx.BadRequest("Invalid file ID", err)
		return httpx.SendResponse(c, response)
	}

	var file models.File
	if err := database.DB.First(&file, fileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response := httpx.NotFound("File not found")
			return httpx.SendResponse(c, response)
		}
		response := httpx.InternalServerError("Failed to fetch file", err)
		return httpx.SendResponse(c, response)
	}

	// Log access
	h.logFileAccess(file.ID, c, "view")

	response := httpx.OK("File retrieved successfully", file)
	return httpx.SendResponse(c, response)
}

// DownloadFile handles file download requests
func (h *FileHandler) DownloadFile(c *fiber.Ctx) error {
	id := c.Params("id")
	fileID, err := uuid.Parse(id)
	if err != nil {
		response := httpx.BadRequest("Invalid file ID", err)
		return httpx.SendResponse(c, response)
	}

	var file models.File
	if err := database.DB.First(&file, fileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response := httpx.NotFound("File not found")
			return httpx.SendResponse(c, response)
		}
		response := httpx.InternalServerError("Failed to fetch file", err)
		return httpx.SendResponse(c, response)
	}

	// Check if file exists on disk
	if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
		response := httpx.NotFound("File not found on disk")
		return httpx.SendResponse(c, response)
	}

	// Log access
	h.logFileAccess(file.ID, c, "download")

	// Update access count and last accessed time
	database.DB.Model(&file).Updates(map[string]interface{}{
		"access_count":  gorm.Expr("access_count + 1"),
		"last_accessed": time.Now(),
	})

	// Send file
	return c.Download(file.FilePath, file.OriginalName)
}

// UpdateFile updates file information
func (h *FileHandler) UpdateFile(c *fiber.Ctx) error {
	id := c.Params("id")
	fileID, err := uuid.Parse(id)
	if err != nil {
		response := httpx.BadRequest("Invalid file ID", err)
		return httpx.SendResponse(c, response)
	}

	var input requests.UpdateFileRequest
	if err := c.BodyParser(&input); err != nil {
		response := httpx.BadRequest("Invalid request body", err)
		return httpx.SendResponse(c, response)
	}

	var file models.File
	if err := database.DB.First(&file, fileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response := httpx.NotFound("File not found")
			return httpx.SendResponse(c, response)
		}
		response := httpx.InternalServerError("Failed to fetch file", err)
		return httpx.SendResponse(c, response)
	}

	// Update fields
	updates := make(map[string]interface{})
	if input.FileName != nil {
		updates["original_name"] = *input.FileName
	}
	if input.Metadata != nil {
		updates["metadata"] = sql.JSONB(input.Metadata)
	}
	if input.ExpiresAt != nil {
		updates["expires_at"] = input.ExpiresAt
	}
	if input.Status != nil {
		updates["status"] = input.Status
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&file).Updates(updates).Error; err != nil {
			response := httpx.InternalServerError("Failed to update file", err)
			return httpx.SendResponse(c, response)
		}
	}

	response := httpx.OK("File updated successfully", file)
	return httpx.SendResponse(c, response)
}

// DeleteFile deletes a file
func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	id := c.Params("id")
	fileID, err := uuid.Parse(id)
	if err != nil {
		response := httpx.BadRequest("Invalid file ID", err)
		return httpx.SendResponse(c, response)
	}

	var file models.File
	if err := database.DB.First(&file, fileID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response := httpx.NotFound("File not found")
			return httpx.SendResponse(c, response)
		}
		response := httpx.InternalServerError("Failed to fetch file", err)
		return httpx.SendResponse(c, response)
	}

	// Use transaction to delete file and related records
	err = sql.WithTransaction(database.DB, func(tx *gorm.DB) error {
		// Delete file shares
		if err := tx.Where("file_id = ?", fileID).Delete(&models.FileShare{}).Error; err != nil {
			return err
		}

		// Delete file access logs
		if err := tx.Where("file_id = ?", fileID).Delete(&models.FileAccess{}).Error; err != nil {
			return err
		}

		// Delete file record
		if err := tx.Delete(&file).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		response := httpx.InternalServerError("Failed to delete file", err)
		return httpx.SendResponse(c, response)
	}

	// Delete file from disk
	if err := os.Remove(file.FilePath); err != nil {
		log.Printf("Warning: Failed to delete file from disk: %v", err)
	}

	response := httpx.OK("File deleted successfully", nil)
	return httpx.SendResponse(c, response)
}

// SearchFiles searches for files based on criteria
func (h *FileHandler) SearchFiles(c *fiber.Ctx) error {
	var input requests.FileSearchRequest
	if err := c.QueryParser(&input); err != nil {
		response := httpx.BadRequest("Invalid query parameters", err)
		return httpx.SendResponse(c, response)
	}

	// Set defaults
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.SortBy == "" {
		input.SortBy = "created_at"
	}
	if input.SortOrder == "" {
		input.SortOrder = "desc"
	}

	// Build query
	query := database.DB.Model(&models.File{})

	// Apply filters
	if input.Query != "" {
		query = query.Where("original_name ILIKE ? OR file_type ILIKE ?", "%"+input.Query+"%", "%"+input.Query+"%")
	}
	if input.FileType != "" {
		query = query.Where("file_type = ?", input.FileType)
	}
	if input.Status != "" {
		query = query.Where("status = ?", input.Status)
	}
	if input.UploadedBy != nil {
		query = query.Where("uploaded_by = ?", input.UploadedBy)
	}
	if input.UploadedAfter != nil {
		query = query.Where("created_at >= ?", input.UploadedAfter)
	}
	if input.UploadedBefore != nil {
		query = query.Where("created_at <= ?", input.UploadedBefore)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response := httpx.InternalServerError("Failed to count files", err)
		return httpx.SendResponse(c, response)
	}

	// Apply sorting and pagination
	offset := (input.Page - 1) * input.Limit
	query = query.Order(input.SortBy + " " + input.SortOrder).
		Offset(offset).
		Limit(input.Limit)

	var files []models.File
	if err := query.Find(&files).Error; err != nil {
		response := httpx.InternalServerError("Failed to fetch files", err)
		return httpx.SendResponse(c, response)
	}

	// Build response
	result := map[string]interface{}{
		"files": files,
		"pagination": map[string]interface{}{
			"page":       input.Page,
			"limit":      input.Limit,
			"total":      total,
			"totalPages": (total + int64(input.Limit) - 1) / int64(input.Limit),
		},
	}

	response := httpx.OK("Files retrieved successfully", result)
	return httpx.SendResponse(c, response)
}

// logFileAccess logs file access for analytics
func (h *FileHandler) logFileAccess(fileID uuid.UUID, c *fiber.Ctx, accessType string) {
	fileAccess := models.FileAccess{
		FileID:     fileID,
		IPAddress:  c.IP(),
		UserAgent:  c.Get("User-Agent"),
		Referer:    c.Get("Referer"),
		AccessType: accessType,
		AccessTime: time.Now(),
		SessionID:  c.Get("X-Session-ID"),
	}

	// Try to get user ID from context if available
	if userID := c.Locals("userID"); userID != nil {
		if uid, ok := userID.(uuid.UUID); ok {
			fileAccess.UserID = &uid
		}
	}

	// Log access asynchronously
	go func() {
		if err := database.DB.Create(&fileAccess).Error; err != nil {
			log.Printf("Failed to log file access: %v", err)
		}
	}()
}
