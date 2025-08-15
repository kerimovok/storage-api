package handlers

import (
	"fmt"
	"log"
	"os"
	"storage-api/internal/database"
	"storage-api/internal/models"
	"storage-api/internal/requests"
	"storage-api/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
	form, err := c.MultipartForm()
	if err != nil {
		response := httpx.BadRequest("Failed to parse multipart form", err)
		return httpx.SendResponse(c, response)
	}

	// Get files from form
	files := form.File["files"]
	if len(files) == 0 {
		response := httpx.BadRequest("No files provided. Use 'files' field for file uploads", nil)
		return httpx.SendResponse(c, response)
	}

	// Validate multiple files
	if err := h.fileService.ValidateMultipleFiles(files); err != nil {
		response := httpx.BadRequest("File validation failed", err)
		return httpx.SendResponse(c, response)
	}

	// Process all files
	uploadResults, err := h.fileService.ProcessMultipleFiles(files)
	if err != nil {
		response := httpx.InternalServerError("Failed to process files", err)
		return httpx.SendResponse(c, response)
	}

	// Create file records for successful uploads
	var fileRecords []models.File
	var failedUploads []map[string]interface{}

	for _, result := range uploadResults {
		if result.Success {
			// Create file record
			fileRecord := models.File{
				OriginalName: result.OriginalName,
				StoredName:   result.StoredName,
				FilePath:     result.FilePath,
				FileSize:     result.FileSize,
				MimeType:     result.MimeType,
				Extension:    result.Extension,
				FileType:     result.FileType,
				Hash:         result.Hash,
				Status:       "active",
			}

			// Save file record
			if err := database.DB.Create(&fileRecord).Error; err != nil {
				log.Printf("Failed to save file record for %s: %v", result.OriginalName, err)
				// Mark as failed
				result.Success = false
				result.Error = "Failed to save file record"
			} else {
				fileRecords = append(fileRecords, fileRecord)
			}
		}

		// Add failed uploads to separate list
		if !result.Success {
			failedUploads = append(failedUploads, map[string]interface{}{
				"original_name": result.OriginalName,
				"error":         result.Error,
			})
		}
	}

	// Build response
	responseData := map[string]interface{}{
		"uploaded_files": fileRecords,
		"total_files":    len(files),
		"successful":     len(fileRecords),
		"failed":         len(failedUploads),
	}

	if len(failedUploads) > 0 {
		responseData["failed_uploads"] = failedUploads
	}

	// Determine response message
	var message string
	if len(failedUploads) == 0 {
		message = "All files uploaded successfully"
	} else if len(fileRecords) == 0 {
		message = "No files were uploaded successfully"
	} else {
		message = fmt.Sprintf("Uploaded %d files, %d failed", len(fileRecords), len(failedUploads))
	}

	// Determine HTTP status
	var status int
	if len(failedUploads) == 0 {
		status = fiber.StatusCreated
	} else if len(fileRecords) == 0 {
		status = fiber.StatusBadRequest
	} else {
		status = fiber.StatusPartialContent
	}

	response := httpx.Response{
		Success: true,
		Message: message,
		Data:    responseData,
		Status:  status,
	}

	return c.Status(status).JSON(response)
}

// GetFile retrieves file information or downloads the file based on query parameter
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

	// Check if download is requested via query parameter
	download := c.Query("download")
	if download == "true" || download == "1" {
		// Check if file exists on disk
		if _, err := os.Stat(file.FilePath); os.IsNotExist(err) {
			response := httpx.NotFound("File not found on disk")
			return httpx.SendResponse(c, response)
		}

		// Send file for download
		return c.Download(file.FilePath, file.OriginalName)
	}

	// Return file metadata by default
	response := httpx.OK("File retrieved successfully", file)
	return httpx.SendResponse(c, response)
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

	// Validate request
	if err := validator.ValidateStruct(&input); err != nil {
		response := httpx.BadRequest("Validation failed", err)
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

	// Delete file record
	if err := database.DB.Delete(&file).Error; err != nil {
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

	// Validate request
	if err := validator.ValidateStruct(&input); err != nil {
		response := httpx.BadRequest("Validation failed", err)
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
	if input.FileType != "" {
		query = query.Where("file_type = ?", input.FileType)
	}
	if input.Status != "" {
		query = query.Where("status = ?", input.Status)
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

// GetFileLimits returns file size limits for different extensions
func (h *FileHandler) GetFileLimits(c *fiber.Ctx) error {
	uploadConfig := h.fileService.GetUploadConfig()

	limits := map[string]interface{}{
		"default_max_size": h.fileService.GetMaxFileSizeForExtension(""),
		"extensions":       make(map[string]int64),
		"upload_limits": map[string]interface{}{
			"max_files":      uploadConfig.MaxFiles,
			"max_total_size": uploadConfig.MaxTotalSize,
		},
	}

	// Get limits for each allowed extension
	for _, ext := range []string{"jpg", "jpeg", "png", "gif", "pdf", "doc", "docx", "xls", "xlsx", "txt", "csv", "zip", "rar", "mp4", "avi", "mov", "mp3", "wav"} {
		limits["extensions"].(map[string]int64)[ext] = h.fileService.GetMaxFileSizeForExtension(ext)
	}

	response := httpx.OK("File limits retrieved successfully", limits)
	return httpx.SendResponse(c, response)
}
