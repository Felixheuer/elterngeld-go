package handlers

import (
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

func NewDocumentHandler(db *gorm.DB, logger *zap.Logger, config *config.Config) *DocumentHandler {
	return &DocumentHandler{
		db:     db,
		logger: logger,
		config: config,
	}
}

// UploadDocumentRequest represents the document upload request
type UploadDocumentRequest struct {
	LeadID    *uuid.UUID `form:"lead_id,omitempty"`
	BookingID *uuid.UUID `form:"booking_id,omitempty"`
	Category  string     `form:"category" binding:"required"`
	IsPublic  bool       `form:"is_public,omitempty"`
	Notes     string     `form:"notes,omitempty"`
}

// ListDocuments handles listing documents with filtering
// @Summary List documents
// @Description Get list of documents with filtering options
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Param category query string false "Filter by category"
// @Param lead_id query string false "Filter by lead ID"
// @Param booking_id query string false "Filter by booking ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/documents [get]
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("user_role")

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	// Parse filters
	category := c.Query("category")
	leadID := c.Query("lead_id")
	bookingID := c.Query("booking_id")

	// Build query
	query := h.db.Model(&models.Document{})

	// Role-based filtering
	if userRole == "user" {
		// Users can only see their own documents
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		// Junior beraters can see documents from assigned leads
		query = query.Joins("LEFT JOIN leads ON documents.lead_id = leads.id").
			Where("documents.user_id = ? OR leads.assigned_to_id = ? OR documents.is_public = ?", 
				userID, userID, true)
	} else if userRole == "berater" {
		// Beraters can see most documents
		query = query.Where("is_public = ? OR user_id = ?", true, userID)
	}
	// Admins can see all documents

	// Apply filters
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if leadID != "" {
		query = query.Where("lead_id = ?", leadID)
	}
	if bookingID != "" {
		query = query.Where("booking_id = ?", bookingID)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get documents with preloaded relations
	var documents []models.Document
	if err := query.Preload("User").Preload("Lead").Preload("Booking").
		Offset(offset).Limit(limit).Order("created_at DESC").Find(&documents).Error; err != nil {
		h.logger.Error("Failed to fetch documents", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"documents": documents,
		"pagination": gin.H{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// UploadDocument handles document upload
// @Summary Upload document
// @Description Upload a document with categorization
// @Tags documents
// @Security BearerAuth
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Document file"
// @Param lead_id formData string false "Lead ID"
// @Param booking_id formData string false "Booking ID"
// @Param category formData string true "Document category"
// @Param is_public formData bool false "Is document public"
// @Param notes formData string false "Document notes"
// @Success 201 {object} models.Document
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/documents [post]
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Parse form data
	var req UploadDocumentRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data", "details": err.Error()})
		return
	}

	// Get uploaded file
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Validate file
	if err := h.validateFile(fileHeader); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify lead/booking exists if provided
	if req.LeadID != nil {
		var lead models.Lead
		if err := h.db.Where("id = ?", *req.LeadID).First(&lead).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lead ID"})
			return
		}
	}

	if req.BookingID != nil {
		var booking models.Booking
		if err := h.db.Where("id = ?", *req.BookingID).First(&booking).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid booking ID"})
			return
		}
	}

	// Generate unique filename
	ext := filepath.Ext(fileHeader.Filename)
	filename := uuid.New().String() + ext
	
	// Store file
	filePath, err := h.storeFile(file, filename)
	if err != nil {
		h.logger.Error("Failed to store file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store file"})
		return
	}

	// Create document record
	document := models.Document{
		ID:            uuid.New(),
		UserID:        userID.(uuid.UUID),
		LeadID:        *req.LeadID,
		FileName:      fileHeader.Filename,
		OriginalName:  fileHeader.Filename,
		FilePath:      filePath,
		FileSize:      fileHeader.Size,
		ContentType:   fileHeader.Header.Get("Content-Type"),
		DocumentType:  models.DocumentTypeOther,
		Description:   req.Notes,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := h.db.Create(&document).Error; err != nil {
		h.logger.Error("Failed to create document record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save document"})
		return
	}

	h.logger.Info("Document uploaded successfully", 
		zap.String("document_id", document.ID.String()),
		zap.String("filename", document.FileName),
		zap.String("user_id", userID.(uuid.UUID).String()))

	c.JSON(http.StatusCreated, document)
}

// GetDocument handles getting a specific document
// @Summary Get document by ID
// @Description Get document information
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} models.Document
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/documents/{id} [get]
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	documentID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var document models.Document
	query := h.db.Where("id = ?", documentID)

	// Role-based access control
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Joins("LEFT JOIN leads ON documents.lead_id = leads.id").
			Where("documents.user_id = ? OR leads.assigned_to_id = ? OR documents.is_public = ?", 
				userID, userID, true)
	} else if userRole == "berater" {
		query = query.Where("is_public = ? OR user_id = ?", true, userID)
	}

	if err := query.Preload("User").Preload("Lead").Preload("Booking").First(&document).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			h.logger.Error("Failed to fetch document", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document"})
		}
		return
	}

	c.JSON(http.StatusOK, document)
}

// DownloadDocument handles document download
// @Summary Download document
// @Description Download document file
// @Tags documents
// @Security BearerAuth
// @Produce application/octet-stream
// @Param id path string true "Document ID"
// @Success 200 {file} file "Document file"
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/documents/{id}/download [get]
func (h *DocumentHandler) DownloadDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	documentID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var document models.Document
	query := h.db.Where("id = ?", documentID)

	// Same access control as GetDocument
	if userRole == "user" {
		query = query.Where("user_id = ?", userID)
	} else if userRole == "junior_berater" {
		query = query.Joins("LEFT JOIN leads ON documents.lead_id = leads.id").
			Where("documents.user_id = ? OR leads.assigned_to_id = ? OR documents.is_public = ?", 
				userID, userID, true)
	} else if userRole == "berater" {
		query = query.Where("is_public = ? OR user_id = ?", true, userID)
	}

	if err := query.First(&document).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document"})
		}
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+document.FileName)
	c.Header("Content-Type", document.ContentType)

	// Serve file
	c.File(document.FilePath)
}

// UpdateDocument handles updating document metadata
// @Summary Update document
// @Description Update document metadata
// @Tags documents
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Document ID"
// @Param request body map[string]interface{} true "Update data"
// @Success 200 {object} models.Document
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/documents/{id} [put]
func (h *DocumentHandler) UpdateDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	documentID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var document models.Document
	query := h.db.Where("id = ?", documentID)

	// Only owner or admin can update
	if userRole != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&document).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document"})
		}
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Only allow certain fields to be updated
	allowedFields := []string{"category", "is_public", "notes"}
	filteredUpdates := make(map[string]interface{})
	for _, field := range allowedFields {
		if value, exists := updates[field]; exists {
			filteredUpdates[field] = value
		}
	}

	if len(filteredUpdates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No valid fields to update"})
		return
	}

	filteredUpdates["updated_at"] = time.Now()

	if err := h.db.Model(&document).Updates(filteredUpdates).Error; err != nil {
		h.logger.Error("Failed to update document", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update document"})
		return
	}

	// Fetch updated document
	if err := h.db.First(&document, "id = ?", documentID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated document"})
		return
	}

	c.JSON(http.StatusOK, document)
}

// DeleteDocument handles deleting a document
// @Summary Delete document
// @Description Delete a document and its file
// @Tags documents
// @Security BearerAuth
// @Produce json
// @Param id path string true "Document ID"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/documents/{id} [delete]
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	documentID := c.Param("id")
	userRole, _ := c.Get("user_role")

	var document models.Document
	query := h.db.Where("id = ?", documentID)

	// Only owner or admin can delete
	if userRole != "admin" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.First(&document).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch document"})
		}
		return
	}

	// Delete database record (soft delete)
	if err := h.db.Delete(&document).Error; err != nil {
		h.logger.Error("Failed to delete document record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	// TODO: Delete actual file from storage
	// For now, we keep the file for data integrity

	h.logger.Info("Document deleted successfully", zap.String("document_id", documentID))

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// validateFile validates uploaded file
func (h *DocumentHandler) validateFile(fileHeader *multipart.FileHeader) error {
	// Check file size (max 10MB)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if fileHeader.Size > maxSize {
		return gin.Error{Err: &gin.Error{Meta: "File size exceeds maximum allowed size (10MB)"}}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowedExts := []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".doc", ".docx", ".txt", ".zip"}
	
	isAllowed := false
	for _, allowedExt := range allowedExts {
		if ext == allowedExt {
			isAllowed = true
			break
		}
	}
	
	if !isAllowed {
		return gin.Error{Err: &gin.Error{Meta: "File type not allowed"}}
	}

	return nil
}

// storeFile stores the uploaded file
func (h *DocumentHandler) storeFile(file multipart.File, filename string) (string, error) {
	// For now, store locally
	// TODO: Implement S3 storage when h.config.S3.UseS3 is true
	
	uploadPath := h.config.Upload.Path
	if uploadPath == "" {
		uploadPath = "./storage/uploads"
	}
	
	filePath := filepath.Join(uploadPath, filename)
	
	// Create file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	
	// Copy file content
	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}
	
	return filePath, nil
}