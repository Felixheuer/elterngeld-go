package models

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentType string

const (
	DocumentTypeBirthCertificate DocumentType = "geburtsurkunde"
	DocumentTypeIncomeProof      DocumentType = "einkommensnachweis"
	DocumentTypeEmploymentCert   DocumentType = "arbeitsbescheinigung"
	DocumentTypeApplication      DocumentType = "antrag"
	DocumentTypeOther            DocumentType = "sonstiges"
)

// Document represents an uploaded file/document
type Document struct {
	ID     uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	LeadID uuid.UUID `json:"lead_id" gorm:"type:char(36);not null;index"`
	UserID uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`

	// File information
	FileName      string `json:"file_name" gorm:"not null" validate:"required"`
	OriginalName  string `json:"original_name" gorm:"not null" validate:"required"`
	FilePath      string `json:"file_path" gorm:"not null" validate:"required"`
	FileSize      int64  `json:"file_size" gorm:"not null"`
	ContentType   string `json:"content_type" gorm:"not null"`
	FileExtension string `json:"file_extension" gorm:"not null"`

	// Document metadata
	DocumentType DocumentType `json:"document_type" gorm:"not null;default:'sonstiges'" validate:"required"`
	Description  string       `json:"description" gorm:"type:text"`
	IsProcessed  bool         `json:"is_processed" gorm:"not null;default:false"`

	// S3 information (if using S3)
	S3Bucket string `json:"s3_bucket" gorm:""`
	S3Key    string `json:"s3_key" gorm:""`
	S3URL    string `json:"s3_url" gorm:""`

	// Timestamps
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	Lead Lead `json:"lead,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User User `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// DocumentResponse represents the document data returned in API responses
type DocumentResponse struct {
	ID            uuid.UUID    `json:"id"`
	LeadID        uuid.UUID    `json:"lead_id"`
	UserID        uuid.UUID    `json:"user_id"`
	FileName      string       `json:"file_name"`
	OriginalName  string       `json:"original_name"`
	FileSize      int64        `json:"file_size"`
	ContentType   string       `json:"content_type"`
	FileExtension string       `json:"file_extension"`
	DocumentType  DocumentType `json:"document_type"`
	Description   string       `json:"description"`
	IsProcessed   bool         `json:"is_processed"`
	DownloadURL   string       `json:"download_url"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// UploadDocumentRequest represents the request for uploading a document
type UploadDocumentRequest struct {
	DocumentType DocumentType `form:"document_type" validate:"required,oneof=geburtsurkunde einkommensnachweis arbeitsbescheinigung antrag sonstiges"`
	Description  string       `form:"description"`
}

// UpdateDocumentRequest represents the request for updating document metadata
type UpdateDocumentRequest struct {
	DocumentType *DocumentType `json:"document_type" validate:"omitempty,oneof=geburtsurkunde einkommensnachweis arbeitsbescheinigung antrag sonstiges"`
	Description  *string       `json:"description"`
	IsProcessed  *bool         `json:"is_processed"`
}

// BeforeCreate is a GORM hook that runs before creating a document
func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}

	// Extract file extension from original name
	if d.FileExtension == "" && d.OriginalName != "" {
		d.FileExtension = strings.ToLower(filepath.Ext(d.OriginalName))
	}

	return nil
}

// ToResponse converts a Document to DocumentResponse
func (d *Document) ToResponse(baseURL string) DocumentResponse {
	downloadURL := ""
	if d.S3URL != "" {
		downloadURL = d.S3URL
	} else if baseURL != "" {
		downloadURL = baseURL + "/api/v1/documents/" + d.ID.String() + "/download"
	}

	return DocumentResponse{
		ID:            d.ID,
		LeadID:        d.LeadID,
		UserID:        d.UserID,
		FileName:      d.FileName,
		OriginalName:  d.OriginalName,
		FileSize:      d.FileSize,
		ContentType:   d.ContentType,
		FileExtension: d.FileExtension,
		DocumentType:  d.DocumentType,
		Description:   d.Description,
		IsProcessed:   d.IsProcessed,
		DownloadURL:   downloadURL,
		CreatedAt:     d.CreatedAt,
		UpdatedAt:     d.UpdatedAt,
	}
}

// IsImage checks if the document is an image
func (d *Document) IsImage() bool {
	imageTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp"}
	for _, imageType := range imageTypes {
		if d.ContentType == imageType {
			return true
		}
	}
	return false
}

// IsPDF checks if the document is a PDF
func (d *Document) IsPDF() bool {
	return d.ContentType == "application/pdf"
}

// IsValid checks if the document has a valid file type
func (d *Document) IsValid() bool {
	validTypes := []string{
		"application/pdf",
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	for _, validType := range validTypes {
		if d.ContentType == validType {
			return true
		}
	}
	return false
}

// GetHumanReadableSize returns the file size in human readable format
func (d *Document) GetHumanReadableSize() string {
	const unit = 1024
	if d.FileSize < unit {
		return fmt.Sprintf("%d B", d.FileSize)
	}

	div, exp := int64(unit), 0
	for n := d.FileSize / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(d.FileSize)/float64(div), "KMGTPE"[exp])
}

// DocumentTypeDisplayName returns the display name for document type
func (dt DocumentType) DisplayName() string {
	switch dt {
	case DocumentTypeBirthCertificate:
		return "Geburtsurkunde"
	case DocumentTypeIncomeProof:
		return "Einkommensnachweis"
	case DocumentTypeEmploymentCert:
		return "Arbeitsbescheinigung"
	case DocumentTypeApplication:
		return "Antrag"
	case DocumentTypeOther:
		return "Sonstiges"
	default:
		return "Unbekannt"
	}
}
