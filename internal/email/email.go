package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"elterngeld-portal/config"
	"elterngeld-portal/internal/models"

	"go.uber.org/zap"
)

type EmailService struct {
	config *config.Config
	logger *zap.Logger
	auth   smtp.Auth
}

type EmailData struct {
	To       []string
	Subject  string
	Template string
	Data     interface{}
}

// Template data structures
type WelcomeEmailData struct {
	Name             string
	Email            string
	VerificationURL  string
	SupportEmail     string
}

type BookingConfirmationData struct {
	Name            string
	BookingRef      string
	PackageName     string
	TotalPrice      float64
	Currency        string
	TimeslotDate    string
	TimeslotTime    string
	OnlineMeetingURL string
	SupportEmail    string
}

type TodoNotificationData struct {
	Name         string
	TodoTitle    string
	TodoDesc     string
	DueDate      string
	Priority     string
	AssignedBy   string
	DashboardURL string
	SupportEmail string
}

type LeadAssignmentData struct {
	BeraterName   string
	LeadTitle     string
	LeadDesc      string
	CustomerName  string
	CustomerEmail string
	Priority      string
	DashboardURL  string
}

type PaymentConfirmationData struct {
	Name         string
	BookingRef   string
	Amount       float64
	Currency     string
	PackageName  string
	PaymentDate  string
	InvoiceURL   string
	SupportEmail string
}

func NewEmailService(config *config.Config, logger *zap.Logger) *EmailService {
	var auth smtp.Auth
	if config.SMTP.Username != "" && config.SMTP.Password != "" {
		auth = smtp.PlainAuth("", config.SMTP.Username, config.SMTP.Password, config.SMTP.Host)
	}

	return &EmailService{
		config: config,
		logger: logger,
		auth:   auth,
	}
}

// SendWelcomeEmail sends welcome email with verification link
func (e *EmailService) SendWelcomeEmail(user *models.User, verificationToken string) error {
	verificationURL := fmt.Sprintf("%s/auth/verify-email?token=%s", e.config.App.BaseURL, verificationToken)
	
	data := WelcomeEmailData{
		Name:             user.FirstName + " " + user.LastName,
		Email:            user.Email,
		VerificationURL:  verificationURL,
		SupportEmail:     e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{user.Email},
		Subject:  "Willkommen beim Elterngeld-Portal - E-Mail bestätigen",
		Template: "welcome",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendBookingConfirmation sends booking confirmation email
func (e *EmailService) SendBookingConfirmation(booking *models.Booking, user *models.User) error {
	var timeslotInfo string
	if booking.Timeslot != nil {
		timeslotInfo = booking.Timeslot.DateTime.Format("02.01.2006 um 15:04")
	}

	data := BookingConfirmationData{
		Name:            user.FirstName + " " + user.LastName,
		BookingRef:      booking.BookingReference,
		PackageName:     booking.Package.Name,
		TotalPrice:      booking.TotalPrice,
		Currency:        booking.Currency,
		TimeslotDate:    timeslotInfo,
		OnlineMeetingURL: booking.OnlineMeetingURL,
		SupportEmail:    e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{user.Email},
		Subject:  fmt.Sprintf("Buchungsbestätigung - %s", booking.BookingReference),
		Template: "booking_confirmation",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendTodoNotification sends todo notification to user
func (e *EmailService) SendTodoNotification(todo *models.Todo, user *models.User, assignedBy *models.User) error {
	var dueDate string
	if todo.DueDate != nil {
		dueDate = todo.DueDate.Format("02.01.2006")
	}

	dashboardURL := fmt.Sprintf("%s/dashboard/todos", e.config.App.BaseURL)

	data := TodoNotificationData{
		Name:         user.FirstName + " " + user.LastName,
		TodoTitle:    todo.Title,
		TodoDesc:     todo.Description,
		DueDate:      dueDate,
		Priority:     todo.Priority,
		AssignedBy:   assignedBy.FirstName + " " + assignedBy.LastName,
		DashboardURL: dashboardURL,
		SupportEmail: e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{user.Email},
		Subject:  fmt.Sprintf("Neue Aufgabe zugewiesen: %s", todo.Title),
		Template: "todo_notification",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendLeadAssignment sends lead assignment notification to berater
func (e *EmailService) SendLeadAssignment(lead *models.Lead, berater *models.User) error {
	dashboardURL := fmt.Sprintf("%s/dashboard/leads/%s", e.config.App.BaseURL, lead.ID.String())

	data := LeadAssignmentData{
		BeraterName:   berater.FirstName + " " + berater.LastName,
		LeadTitle:     lead.Title,
		LeadDesc:      lead.Description,
		CustomerName:  lead.ContactEmail, // Use email if no name available
		CustomerEmail: lead.ContactEmail,
		Priority:      string(lead.Priority),
		DashboardURL:  dashboardURL,
	}

	if lead.User != nil {
		data.CustomerName = lead.User.FirstName + " " + lead.User.LastName
	}

	emailData := EmailData{
		To:       []string{berater.Email},
		Subject:  fmt.Sprintf("Neuer Lead zugewiesen: %s", lead.Title),
		Template: "lead_assignment",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendPaymentConfirmation sends payment confirmation email
func (e *EmailService) SendPaymentConfirmation(payment *models.Payment, booking *models.Booking, user *models.User) error {
	data := PaymentConfirmationData{
		Name:         user.FirstName + " " + user.LastName,
		BookingRef:   booking.BookingReference,
		Amount:       payment.Amount,
		Currency:     payment.Currency,
		PackageName:  booking.Package.Name,
		PaymentDate:  payment.CompletedAt.Format("02.01.2006"),
		SupportEmail: e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{user.Email},
		Subject:  fmt.Sprintf("Zahlungsbestätigung - %s", booking.BookingReference),
		Template: "payment_confirmation",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendPasswordReset sends password reset email
func (e *EmailService) SendPasswordReset(user *models.User, resetToken string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s", e.config.App.BaseURL, resetToken)
	
	data := map[string]interface{}{
		"Name":         user.FirstName + " " + user.LastName,
		"ResetURL":     resetURL,
		"SupportEmail": e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{user.Email},
		Subject:  "Passwort zurücksetzen - Elterngeld-Portal",
		Template: "password_reset",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// SendContactFormConfirmation sends confirmation for contact form submission
func (e *EmailService) SendContactFormConfirmation(contactForm *models.ContactForm) error {
	data := map[string]interface{}{
		"Name":            contactForm.Name,
		"Subject":         contactForm.Subject,
		"ReferenceNumber": "CF-" + contactForm.ID.String()[:8],
		"SupportEmail":    e.config.SMTP.FromEmail,
	}

	emailData := EmailData{
		To:       []string{contactForm.Email},
		Subject:  "Kontaktanfrage erhalten - Elterngeld-Portal",
		Template: "contact_confirmation",
		Data:     data,
	}

	return e.sendEmail(emailData)
}

// sendEmail sends an email using the configured SMTP settings
func (e *EmailService) sendEmail(emailData EmailData) error {
	// In development mode, just log the email instead of sending
	if e.config.IsDevelopment() {
		e.logger.Info("Email would be sent in production",
			zap.Strings("to", emailData.To),
			zap.String("subject", emailData.Subject),
			zap.String("template", emailData.Template))
		return nil
	}

	// Load and parse template
	body, err := e.renderTemplate(emailData.Template, emailData.Data)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Prepare email message
	message := e.buildMessage(emailData.To, emailData.Subject, body)

	// Send email
	addr := fmt.Sprintf("%s:%d", e.config.SMTP.Host, e.config.SMTP.Port)
	to := emailData.To

	err = smtp.SendMail(addr, e.auth, e.config.SMTP.FromEmail, to, []byte(message))
	if err != nil {
		e.logger.Error("Failed to send email", 
			zap.Error(err),
			zap.Strings("to", to),
			zap.String("subject", emailData.Subject))
		return fmt.Errorf("failed to send email: %w", err)
	}

	e.logger.Info("Email sent successfully",
		zap.Strings("to", to),
		zap.String("subject", emailData.Subject))

	return nil
}

// renderTemplate renders an email template with the provided data
func (e *EmailService) renderTemplate(templateName string, data interface{}) (string, error) {
	// Define email templates inline for simplicity
	// In production, these would be loaded from files
	templates := map[string]string{
		"welcome": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Willkommen</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Willkommen beim Elterngeld-Portal!</h1>
        <p>Hallo {{.Name}},</p>
        <p>vielen Dank für Ihre Registrierung beim Elterngeld-Portal. Um Ihr Konto zu aktivieren, bestätigen Sie bitte Ihre E-Mail-Adresse:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationURL}}" style="background-color: #2c5aa0; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">E-Mail bestätigen</a>
        </div>
        <p>Falls Sie Fragen haben, können Sie uns jederzeit unter {{.SupportEmail}} kontaktieren.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"booking_confirmation": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Buchungsbestätigung</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Buchungsbestätigung</h1>
        <p>Hallo {{.Name}},</p>
        <p>Ihre Buchung wurde erfolgreich bestätigt!</p>
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h3>Buchungsdetails:</h3>
            <p><strong>Buchungsnummer:</strong> {{.BookingRef}}</p>
            <p><strong>Paket:</strong> {{.PackageName}}</p>
            <p><strong>Betrag:</strong> {{.TotalPrice}} {{.Currency}}</p>
            {{if .TimeslotDate}}<p><strong>Termin:</strong> {{.TimeslotDate}}</p>{{end}}
            {{if .OnlineMeetingURL}}<p><strong>Online-Meeting:</strong> <a href="{{.OnlineMeetingURL}}">Zum Meeting</a></p>{{end}}
        </div>
        <p>Bei Fragen erreichen Sie uns unter {{.SupportEmail}}.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"todo_notification": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Neue Aufgabe</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Neue Aufgabe zugewiesen</h1>
        <p>Hallo {{.Name}},</p>
        <p>{{.AssignedBy}} hat Ihnen eine neue Aufgabe zugewiesen:</p>
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h3>{{.TodoTitle}}</h3>
            <p>{{.TodoDesc}}</p>
            {{if .DueDate}}<p><strong>Fällig am:</strong> {{.DueDate}}</p>{{end}}
            <p><strong>Priorität:</strong> {{.Priority}}</p>
        </div>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.DashboardURL}}" style="background-color: #2c5aa0; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">Zum Dashboard</a>
        </div>
        <p>Bei Fragen erreichen Sie uns unter {{.SupportEmail}}.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"lead_assignment": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Neuer Lead zugewiesen</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Neuer Lead zugewiesen</h1>
        <p>Hallo {{.BeraterName}},</p>
        <p>Ihnen wurde ein neuer Lead zugewiesen:</p>
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h3>{{.LeadTitle}}</h3>
            <p>{{.LeadDesc}}</p>
            <p><strong>Kunde:</strong> {{.CustomerName}} ({{.CustomerEmail}})</p>
            <p><strong>Priorität:</strong> {{.Priority}}</p>
        </div>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.DashboardURL}}" style="background-color: #2c5aa0; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">Lead bearbeiten</a>
        </div>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"payment_confirmation": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Zahlungsbestätigung</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Zahlungsbestätigung</h1>
        <p>Hallo {{.Name}},</p>
        <p>Ihre Zahlung wurde erfolgreich verarbeitet!</p>
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <h3>Zahlungsdetails:</h3>
            <p><strong>Buchungsnummer:</strong> {{.BookingRef}}</p>
            <p><strong>Paket:</strong> {{.PackageName}}</p>
            <p><strong>Betrag:</strong> {{.Amount}} {{.Currency}}</p>
            <p><strong>Zahlungsdatum:</strong> {{.PaymentDate}}</p>
        </div>
        <p>Bei Fragen erreichen Sie uns unter {{.SupportEmail}}.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"password_reset": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Passwort zurücksetzen</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Passwort zurücksetzen</h1>
        <p>Hallo {{.Name}},</p>
        <p>Sie haben eine Anfrage zum Zurücksetzen Ihres Passworts gestellt. Klicken Sie auf den folgenden Link, um ein neues Passwort zu erstellen:</p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.ResetURL}}" style="background-color: #2c5aa0; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">Passwort zurücksetzen</a>
        </div>
        <p>Dieser Link ist 1 Stunde gültig. Falls Sie diese Anfrage nicht gestellt haben, ignorieren Sie diese E-Mail.</p>
        <p>Bei Fragen erreichen Sie uns unter {{.SupportEmail}}.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,

		"contact_confirmation": `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Kontaktanfrage erhalten</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #2c5aa0;">Kontaktanfrage erhalten</h1>
        <p>Hallo {{.Name}},</p>
        <p>vielen Dank für Ihre Kontaktanfrage. Wir haben Ihre Nachricht erhalten und werden uns schnellstmöglich bei Ihnen melden.</p>
        <div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin: 20px 0;">
            <p><strong>Ihre Anfrage:</strong> {{.Subject}}</p>
            <p><strong>Referenznummer:</strong> {{.ReferenceNumber}}</p>
        </div>
        <p>Bei dringenden Fragen erreichen Sie uns unter {{.SupportEmail}}.</p>
        <p>Ihr Elterngeld-Portal Team</p>
    </div>
</body>
</html>`,
	}

	templateStr, exists := templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	// Parse and execute template
	tmpl, err := template.New(templateName).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// buildMessage builds the email message with headers
func (e *EmailService) buildMessage(to []string, subject, body string) string {
	headers := make(map[string]string)
	headers["From"] = e.config.SMTP.FromEmail
	headers["To"] = strings.Join(to, ", ")
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	return message
}