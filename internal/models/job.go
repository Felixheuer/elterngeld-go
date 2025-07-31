package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type JobStatus string

const (
	JobStatusDraft     JobStatus = "draft"
	JobStatusPublished JobStatus = "published"
	JobStatusPaused    JobStatus = "paused"
	JobStatusClosed    JobStatus = "closed"
	JobStatusArchived  JobStatus = "archived"
)

type JobType string

const (
	JobTypeFullTime   JobType = "full_time"
	JobTypePartTime   JobType = "part_time"
	JobTypeContract   JobType = "contract"
	JobTypeInternship JobType = "internship"
	JobTypeFreelance  JobType = "freelance"
)

type JobLevel string

const (
	JobLevelEntry  JobLevel = "entry"
	JobLevelJunior JobLevel = "junior"
	JobLevelMid    JobLevel = "mid"
	JobLevelSenior JobLevel = "senior"
	JobLevelLead   JobLevel = "lead"
)

type WorkLocation string

const (
	WorkLocationRemote   WorkLocation = "remote"
	WorkLocationOnSite   WorkLocation = "on_site"
	WorkLocationHybrid   WorkLocation = "hybrid"
)

type ApplicationStatus string

const (
	ApplicationStatusSubmitted ApplicationStatus = "submitted"
	ApplicationStatusReviewing ApplicationStatus = "reviewing"
	ApplicationStatusScreening ApplicationStatus = "screening"
	ApplicationStatusInterview ApplicationStatus = "interview"
	ApplicationStatusOffered   ApplicationStatus = "offered"
	ApplicationStatusAccepted  ApplicationStatus = "accepted"
	ApplicationStatusRejected  ApplicationStatus = "rejected"
	ApplicationStatusWithdrawn ApplicationStatus = "withdrawn"
)

// Job represents a job posting
type Job struct {
	ID             uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	Title          string    `json:"title" gorm:"not null" validate:"required"`
	Slug           string    `json:"slug" gorm:"not null;uniqueIndex"`
	Description    string    `json:"description" gorm:"type:text;not null" validate:"required"`
	ShortDescription string  `json:"short_description" gorm:"type:text"`
	
	// Job details
	Status         JobStatus     `json:"status" gorm:"not null;default:'draft'"`
	Type           JobType       `json:"type" gorm:"not null" validate:"required"`
	Level          JobLevel      `json:"level" gorm:"not null" validate:"required"`
	Department     string        `json:"department" gorm:""`
	Location       string        `json:"location" gorm:"not null" validate:"required"`
	WorkLocation   WorkLocation  `json:"work_location" gorm:"not null;default:'on_site'"`
	IsRemote       bool          `json:"is_remote" gorm:"not null;default:false"`
	
	// Compensation
	SalaryMin      *float64 `json:"salary_min" gorm:""`
	SalaryMax      *float64 `json:"salary_max" gorm:""`
	SalaryCurrency string   `json:"salary_currency" gorm:"default:'EUR'"`
	SalaryPeriod   string   `json:"salary_period" gorm:"default:'yearly'"` // yearly, monthly, hourly
	BenefitsText   string   `json:"benefits_text" gorm:"type:text"`
	
	// Requirements
	RequiredSkills     string `json:"required_skills" gorm:"type:text"`     // JSON array
	PreferredSkills    string `json:"preferred_skills" gorm:"type:text"`   // JSON array
	RequiredExperience string `json:"required_experience" gorm:"type:text"`
	EducationRequired  string `json:"education_required" gorm:"type:text"`
	LanguageRequirements string `json:"language_requirements" gorm:"type:text"`
	
	// Application settings
	ApplicationDeadline *time.Time `json:"application_deadline" gorm:""`
	ContactEmail        string     `json:"contact_email" gorm:""`
	ApplicationURL      string     `json:"application_url" gorm:""`
	AllowDirectApply    bool       `json:"allow_direct_apply" gorm:"not null;default:true"`
	
	// SEO and metadata
	MetaTitle       string `json:"meta_title" gorm:""`
	MetaDescription string `json:"meta_description" gorm:"type:text"`
	Tags            string `json:"tags" gorm:"type:text"` // JSON array
	
	// Tracking
	ViewCount        int `json:"view_count" gorm:"default:0"`
	ApplicationCount int `json:"application_count" gorm:"default:0"`
	
	// Publication
	PublishedAt *time.Time `json:"published_at" gorm:""`
	ExpiresAt   *time.Time `json:"expires_at" gorm:""`
	
	// Management
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:char(36);not null;index"`
	UpdatedBy *uuid.UUID `json:"updated_by" gorm:"type:char(36);index"`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Creator      User              `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Updater      *User             `json:"updater,omitempty" gorm:"foreignKey:UpdatedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Applications []JobApplication  `json:"applications,omitempty" gorm:"foreignKey:JobID"`
}

// JobApplication represents a job application
type JobApplication struct {
	ID    uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	JobID uuid.UUID `json:"job_id" gorm:"type:char(36);not null;index"`
	
	// Applicant information
	FirstName   string `json:"first_name" gorm:"not null" validate:"required"`
	LastName    string `json:"last_name" gorm:"not null" validate:"required"`
	Email       string `json:"email" gorm:"not null" validate:"required,email"`
	Phone       string `json:"phone" gorm:""`
	Location    string `json:"location" gorm:""`
	
	// Application details
	Status        ApplicationStatus `json:"status" gorm:"not null;default:'submitted'"`
	CoverLetter   string           `json:"cover_letter" gorm:"type:text"`
	ResumeURL     string           `json:"resume_url" gorm:""`
	PortfolioURL  string           `json:"portfolio_url" gorm:""`
	LinkedInURL   string           `json:"linkedin_url" gorm:""`
	GitHubURL     string           `json:"github_url" gorm:""`
	WebsiteURL    string           `json:"website_url" gorm:""`
	
	// Experience and skills
	YearsExperience   int    `json:"years_experience" gorm:"default:0"`
	CurrentPosition   string `json:"current_position" gorm:""`
	CurrentCompany    string `json:"current_company" gorm:""`
	ExpectedSalary    *float64 `json:"expected_salary" gorm:""`
	AvailabilityDate  *time.Time `json:"availability_date" gorm:""`
	NoticePeriod      string   `json:"notice_period" gorm:""`
	
	// Additional information
	MotivationText    string `json:"motivation_text" gorm:"type:text"`
	Questions         string `json:"questions" gorm:"type:text"`
	PrivacyConsent    bool   `json:"privacy_consent" gorm:"not null;default:false"`
	NewsletterConsent bool   `json:"newsletter_consent" gorm:"not null;default:false"`
	
	// Tracking and source
	Source         string `json:"source" gorm:"default:'website'"` // website, linkedin, referral, etc.
	SourceDetails  string `json:"source_details" gorm:""`
	ReferralName   string `json:"referral_name" gorm:""`
	UtmSource      string `json:"utm_source" gorm:""`
	UtmMedium      string `json:"utm_medium" gorm:""`
	UtmCampaign    string `json:"utm_campaign" gorm:""`
	
	// Review process
	ReviewedBy    *uuid.UUID `json:"reviewed_by" gorm:"type:char(36);index"`
	ReviewedAt    *time.Time `json:"reviewed_at" gorm:""`
	ReviewNotes   string     `json:"review_notes" gorm:"type:text"`
	RejectionNote string     `json:"rejection_note" gorm:"type:text"`
	
	// Communication tracking
	LastContactAt     *time.Time `json:"last_contact_at" gorm:""`
	NextFollowUpAt    *time.Time `json:"next_follow_up_at" gorm:""`
	InterviewScheduled bool      `json:"interview_scheduled" gorm:"not null;default:false"`
	InterviewDate     *time.Time `json:"interview_date" gorm:""`
	
	CreatedAt time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Job      Job   `json:"job,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Reviewer *User `json:"reviewer,omitempty" gorm:"foreignKey:ReviewedBy;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Documents []JobApplicationDocument `json:"documents,omitempty" gorm:"foreignKey:ApplicationID"`
	Activities []JobApplicationActivity `json:"activities,omitempty" gorm:"foreignKey:ApplicationID"`
}

// JobApplicationDocument represents documents attached to job applications
type JobApplicationDocument struct {
	ID            uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	ApplicationID uuid.UUID `json:"application_id" gorm:"type:char(36);not null;index"`
	
	FileName     string `json:"file_name" gorm:"not null"`
	FileSize     int64  `json:"file_size" gorm:"not null"`
	FileType     string `json:"file_type" gorm:"not null"`
	FilePath     string `json:"file_path" gorm:"not null"`
	DocumentType string `json:"document_type" gorm:"not null"` // resume, cover_letter, portfolio, etc.
	
	UploadedAt time.Time      `json:"uploaded_at" gorm:"not null"`
	CreatedAt  time.Time      `json:"created_at" gorm:"not null"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"not null"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	
	// Relationships
	Application JobApplication `json:"application,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

// JobApplicationActivity represents activities/timeline for job applications
type JobApplicationActivity struct {
	ID            uuid.UUID `json:"id" gorm:"type:char(36);primary_key"`
	ApplicationID uuid.UUID `json:"application_id" gorm:"type:char(36);not null;index"`
	UserID        *uuid.UUID `json:"user_id" gorm:"type:char(36);index"`
	
	Type        string `json:"type" gorm:"not null"` // status_change, note_added, email_sent, etc.
	Description string `json:"description" gorm:"not null"`
	Details     string `json:"details" gorm:"type:text"` // JSON data
	
	OldValue string `json:"old_value" gorm:""`
	NewValue string `json:"new_value" gorm:""`
	
	CreatedAt time.Time `json:"created_at" gorm:"not null"`
	
	// Relationships
	Application JobApplication `json:"application,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	User        *User         `json:"user,omitempty" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// Response DTOs
type JobResponse struct {
	ID             uuid.UUID     `json:"id"`
	Title          string        `json:"title"`
	Slug           string        `json:"slug"`
	Description    string        `json:"description"`
	ShortDescription string      `json:"short_description"`
	Status         JobStatus     `json:"status"`
	Type           JobType       `json:"type"`
	Level          JobLevel      `json:"level"`
	Department     string        `json:"department"`
	Location       string        `json:"location"`
	WorkLocation   WorkLocation  `json:"work_location"`
	IsRemote       bool          `json:"is_remote"`
	SalaryMin      *float64      `json:"salary_min"`
	SalaryMax      *float64      `json:"salary_max"`
	SalaryCurrency string        `json:"salary_currency"`
	SalaryPeriod   string        `json:"salary_period"`
	FormattedSalary string       `json:"formatted_salary"`
	BenefitsText   string        `json:"benefits_text"`
	RequiredSkills []string      `json:"required_skills"`
	PreferredSkills []string     `json:"preferred_skills"`
	RequiredExperience string    `json:"required_experience"`
	ApplicationDeadline *time.Time `json:"application_deadline"`
	ContactEmail   string        `json:"contact_email"`
	ApplicationURL string        `json:"application_url"`
	AllowDirectApply bool        `json:"allow_direct_apply"`
	Tags           []string      `json:"tags"`
	ViewCount      int           `json:"view_count"`
	ApplicationCount int         `json:"application_count"`
	PublishedAt    *time.Time    `json:"published_at"`
	ExpiresAt      *time.Time    `json:"expires_at"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Creator        *UserResponse `json:"creator,omitempty"`
	IsExpired      bool          `json:"is_expired"`
	CanApply       bool          `json:"can_apply"`
}

type JobApplicationResponse struct {
	ID              uuid.UUID         `json:"id"`
	JobID           uuid.UUID         `json:"job_id"`
	FirstName       string            `json:"first_name"`
	LastName        string            `json:"last_name"`
	FullName        string            `json:"full_name"`
	Email           string            `json:"email"`
	Phone           string            `json:"phone"`
	Location        string            `json:"location"`
	Status          ApplicationStatus `json:"status"`
	StatusDisplay   string            `json:"status_display"`
	CoverLetter     string            `json:"cover_letter"`
	ResumeURL       string            `json:"resume_url"`
	PortfolioURL    string            `json:"portfolio_url"`
	LinkedInURL     string            `json:"linkedin_url"`
	YearsExperience int               `json:"years_experience"`
	CurrentPosition string            `json:"current_position"`
	CurrentCompany  string            `json:"current_company"`
	ExpectedSalary  *float64          `json:"expected_salary"`
	AvailabilityDate *time.Time       `json:"availability_date"`
	Source          string            `json:"source"`
	ReviewedBy      *UserResponse     `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time        `json:"reviewed_at"`
	ReviewNotes     string            `json:"review_notes"`
	LastContactAt   *time.Time        `json:"last_contact_at"`
	NextFollowUpAt  *time.Time        `json:"next_follow_up_at"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Job             *JobResponse      `json:"job,omitempty"`
	DocumentCount   int               `json:"document_count"`
}

// Request DTOs
type CreateJobRequest struct {
	Title          string       `json:"title" validate:"required"`
	Description    string       `json:"description" validate:"required"`
	ShortDescription string     `json:"short_description"`
	Type           JobType      `json:"type" validate:"required,oneof=full_time part_time contract internship freelance"`
	Level          JobLevel     `json:"level" validate:"required,oneof=entry junior mid senior lead"`
	Department     string       `json:"department"`
	Location       string       `json:"location" validate:"required"`
	WorkLocation   WorkLocation `json:"work_location" validate:"required,oneof=remote on_site hybrid"`
	SalaryMin      *float64     `json:"salary_min"`
	SalaryMax      *float64     `json:"salary_max"`
	SalaryCurrency string       `json:"salary_currency"`
	SalaryPeriod   string       `json:"salary_period"`
	BenefitsText   string       `json:"benefits_text"`
	RequiredSkills []string     `json:"required_skills"`
	PreferredSkills []string    `json:"preferred_skills"`
	RequiredExperience string   `json:"required_experience"`
	EducationRequired string    `json:"education_required"`
	ApplicationDeadline *time.Time `json:"application_deadline"`
	ContactEmail   string       `json:"contact_email"`
	ApplicationURL string       `json:"application_url"`
	AllowDirectApply bool       `json:"allow_direct_apply"`
	Tags           []string     `json:"tags"`
	ExpiresAt      *time.Time   `json:"expires_at"`
}

type UpdateJobRequest struct {
	Title          *string       `json:"title"`
	Description    *string       `json:"description"`
	ShortDescription *string     `json:"short_description"`
	Status         *JobStatus    `json:"status" validate:"omitempty,oneof=draft published paused closed archived"`
	Type           *JobType      `json:"type" validate:"omitempty,oneof=full_time part_time contract internship freelance"`
	Level          *JobLevel     `json:"level" validate:"omitempty,oneof=entry junior mid senior lead"`
	Department     *string       `json:"department"`
	Location       *string       `json:"location"`
	WorkLocation   *WorkLocation `json:"work_location" validate:"omitempty,oneof=remote on_site hybrid"`
	SalaryMin      *float64      `json:"salary_min"`
	SalaryMax      *float64      `json:"salary_max"`
	SalaryCurrency *string       `json:"salary_currency"`
	BenefitsText   *string       `json:"benefits_text"`
	RequiredSkills []string      `json:"required_skills"`
	PreferredSkills []string     `json:"preferred_skills"`
	RequiredExperience *string   `json:"required_experience"`
	ApplicationDeadline *time.Time `json:"application_deadline"`
	ContactEmail   *string       `json:"contact_email"`
	ApplicationURL *string       `json:"application_url"`
	AllowDirectApply *bool       `json:"allow_direct_apply"`
	Tags           []string      `json:"tags"`
	ExpiresAt      *time.Time    `json:"expires_at"`
}

type CreateJobApplicationRequest struct {
	JobID         uuid.UUID  `json:"job_id" validate:"required"`
	FirstName     string     `json:"first_name" validate:"required"`
	LastName      string     `json:"last_name" validate:"required"`
	Email         string     `json:"email" validate:"required,email"`
	Phone         string     `json:"phone"`
	Location      string     `json:"location"`
	CoverLetter   string     `json:"cover_letter"`
	PortfolioURL  string     `json:"portfolio_url"`
	LinkedInURL   string     `json:"linkedin_url"`
	GitHubURL     string     `json:"github_url"`
	WebsiteURL    string     `json:"website_url"`
	YearsExperience int      `json:"years_experience"`
	CurrentPosition string   `json:"current_position"`
	CurrentCompany string    `json:"current_company"`
	ExpectedSalary *float64  `json:"expected_salary"`
	AvailabilityDate *time.Time `json:"availability_date"`
	NoticePeriod  string     `json:"notice_period"`
	MotivationText string    `json:"motivation_text"`
	Questions     string     `json:"questions"`
	PrivacyConsent bool      `json:"privacy_consent" validate:"required"`
	NewsletterConsent bool   `json:"newsletter_consent"`
	Source        string     `json:"source"`
	ReferralName  string     `json:"referral_name"`
}

type UpdateJobApplicationStatusRequest struct {
	Status      ApplicationStatus `json:"status" validate:"required,oneof=submitted reviewing screening interview offered accepted rejected withdrawn"`
	ReviewNotes string           `json:"review_notes"`
	RejectionNote string         `json:"rejection_note"`
}

// BeforeCreate hooks
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	if j.Slug == "" {
		j.Slug = j.generateSlug()
	}
	return nil
}

func (ja *JobApplication) BeforeCreate(tx *gorm.DB) error {
	if ja.ID == uuid.Nil {
		ja.ID = uuid.New()
	}
	return nil
}

func (jad *JobApplicationDocument) BeforeCreate(tx *gorm.DB) error {
	if jad.ID == uuid.Nil {
		jad.ID = uuid.New()
	}
	return nil
}

func (jaa *JobApplicationActivity) BeforeCreate(tx *gorm.DB) error {
	if jaa.ID == uuid.Nil {
		jaa.ID = uuid.New()
	}
	return nil
}

// Helper methods
func (j *Job) generateSlug() string {
	// This would generate a URL-friendly slug from the title
	// For now, just use a simple implementation
	return strings.ToLower(strings.ReplaceAll(j.Title, " ", "-"))
}

func (j *Job) ToResponse() JobResponse {
	response := JobResponse{
		ID:             j.ID,
		Title:          j.Title,
		Slug:           j.Slug,
		Description:    j.Description,
		ShortDescription: j.ShortDescription,
		Status:         j.Status,
		Type:           j.Type,
		Level:          j.Level,
		Department:     j.Department,
		Location:       j.Location,
		WorkLocation:   j.WorkLocation,
		IsRemote:       j.IsRemote,
		SalaryMin:      j.SalaryMin,
		SalaryMax:      j.SalaryMax,
		SalaryCurrency: j.SalaryCurrency,
		SalaryPeriod:   j.SalaryPeriod,
		FormattedSalary: j.FormatSalary(),
		BenefitsText:   j.BenefitsText,
		RequiredSkills: j.GetRequiredSkillsArray(),
		PreferredSkills: j.GetPreferredSkillsArray(),
		RequiredExperience: j.RequiredExperience,
		ApplicationDeadline: j.ApplicationDeadline,
		ContactEmail:   j.ContactEmail,
		ApplicationURL: j.ApplicationURL,
		AllowDirectApply: j.AllowDirectApply,
		Tags:           j.GetTagsArray(),
		ViewCount:      j.ViewCount,
		ApplicationCount: j.ApplicationCount,
		PublishedAt:    j.PublishedAt,
		ExpiresAt:      j.ExpiresAt,
		CreatedAt:      j.CreatedAt,
		UpdatedAt:      j.UpdatedAt,
		IsExpired:      j.IsExpired(),
		CanApply:       j.CanApply(),
	}
	
	if j.Creator.ID != uuid.Nil {
		creatorResponse := j.Creator.ToResponse()
		response.Creator = &creatorResponse
	}
	
	return response
}

func (ja *JobApplication) ToResponse() JobApplicationResponse {
	response := JobApplicationResponse{
		ID:              ja.ID,
		JobID:           ja.JobID,
		FirstName:       ja.FirstName,
		LastName:        ja.LastName,
		FullName:        ja.FirstName + " " + ja.LastName,
		Email:           ja.Email,
		Phone:           ja.Phone,
		Location:        ja.Location,
		Status:          ja.Status,
		StatusDisplay:   ja.Status.GetDisplayName(),
		CoverLetter:     ja.CoverLetter,
		ResumeURL:       ja.ResumeURL,
		PortfolioURL:    ja.PortfolioURL,
		LinkedInURL:     ja.LinkedInURL,
		YearsExperience: ja.YearsExperience,
		CurrentPosition: ja.CurrentPosition,
		CurrentCompany:  ja.CurrentCompany,
		ExpectedSalary:  ja.ExpectedSalary,
		AvailabilityDate: ja.AvailabilityDate,
		Source:          ja.Source,
		ReviewedAt:      ja.ReviewedAt,
		ReviewNotes:     ja.ReviewNotes,
		LastContactAt:   ja.LastContactAt,
		NextFollowUpAt:  ja.NextFollowUpAt,
		CreatedAt:       ja.CreatedAt,
		UpdatedAt:       ja.UpdatedAt,
		DocumentCount:   len(ja.Documents),
	}
	
	if ja.Reviewer != nil && ja.Reviewer.ID != uuid.Nil {
		reviewerResponse := ja.Reviewer.ToResponse()
		response.ReviewedBy = &reviewerResponse
	}
	
	if ja.Job.ID != uuid.Nil {
		jobResponse := ja.Job.ToResponse()
		response.Job = &jobResponse
	}
	
	return response
}

// Utility methods
func (j *Job) FormatSalary() string {
	if j.SalaryMin == nil && j.SalaryMax == nil {
		return ""
	}
	
	if j.SalaryMin != nil && j.SalaryMax != nil {
		return fmt.Sprintf("%.0f - %.0f %s", *j.SalaryMin, *j.SalaryMax, j.SalaryCurrency)
	}
	
	if j.SalaryMin != nil {
		return fmt.Sprintf("ab %.0f %s", *j.SalaryMin, j.SalaryCurrency)
	}
	
	return fmt.Sprintf("bis %.0f %s", *j.SalaryMax, j.SalaryCurrency)
}

func (j *Job) GetRequiredSkillsArray() []string {
	// This would parse JSON array, returning empty for now
	return []string{}
}

func (j *Job) GetPreferredSkillsArray() []string {
	// This would parse JSON array, returning empty for now
	return []string{}
}

func (j *Job) GetTagsArray() []string {
	// This would parse JSON array, returning empty for now
	return []string{}
}

func (j *Job) IsExpired() bool {
	if j.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*j.ExpiresAt)
}

func (j *Job) CanApply() bool {
	if j.Status != JobStatusPublished {
		return false
	}
	if j.IsExpired() {
		return false
	}
	if j.ApplicationDeadline != nil && time.Now().After(*j.ApplicationDeadline) {
		return false
	}
	return j.AllowDirectApply
}

func (j *Job) IsPublished() bool {
	return j.Status == JobStatusPublished && j.PublishedAt != nil
}

func (j *Job) IncrementViewCount() {
	j.ViewCount++
}

func (j *Job) IncrementApplicationCount() {
	j.ApplicationCount++
}

// Status helper methods
func (js JobStatus) GetDisplayName() string {
	switch js {
	case JobStatusDraft:
		return "Entwurf"
	case JobStatusPublished:
		return "Veröffentlicht"
	case JobStatusPaused:
		return "Pausiert"
	case JobStatusClosed:
		return "Geschlossen"
	case JobStatusArchived:
		return "Archiviert"
	default:
		return "Unbekannt"
	}
}

func (jt JobType) GetDisplayName() string {
	switch jt {
	case JobTypeFullTime:
		return "Vollzeit"
	case JobTypePartTime:
		return "Teilzeit"
	case JobTypeContract:
		return "Vertrag"
	case JobTypeInternship:
		return "Praktikum"
	case JobTypeFreelance:
		return "Freiberuflich"
	default:
		return "Unbekannt"
	}
}

func (jl JobLevel) GetDisplayName() string {
	switch jl {
	case JobLevelEntry:
		return "Einsteiger"
	case JobLevelJunior:
		return "Junior"
	case JobLevelMid:
		return "Middle"
	case JobLevelSenior:
		return "Senior"
	case JobLevelLead:
		return "Lead"
	default:
		return "Unbekannt"
	}
}

func (wl WorkLocation) GetDisplayName() string {
	switch wl {
	case WorkLocationRemote:
		return "Remote"
	case WorkLocationOnSite:
		return "Vor Ort"
	case WorkLocationHybrid:
		return "Hybrid"
	default:
		return "Unbekannt"
	}
}

func (as ApplicationStatus) GetDisplayName() string {
	switch as {
	case ApplicationStatusSubmitted:
		return "Eingereicht"
	case ApplicationStatusReviewing:
		return "In Prüfung"
	case ApplicationStatusScreening:
		return "Screening"
	case ApplicationStatusInterview:
		return "Interview"
	case ApplicationStatusOffered:
		return "Angebot erhalten"
	case ApplicationStatusAccepted:
		return "Angenommen"
	case ApplicationStatusRejected:
		return "Abgelehnt"
	case ApplicationStatusWithdrawn:
		return "Zurückgezogen"
	default:
		return "Unbekannt"
	}
}