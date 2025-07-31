-- Initial schema migration for Elterngeld Portal CRM System
-- This migration creates all necessary tables for the complete business plan

-- Users table (extended)
CREATE TABLE IF NOT EXISTS users (
    id CHAR(36) PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    role ENUM('user', 'berater', 'junior_berater', 'admin') NOT NULL DEFAULT 'user',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Profile information
    date_of_birth DATETIME,
    address TEXT,
    postal_code VARCHAR(20),
    city VARCHAR(255),
    
    -- Email verification
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    email_verified_at DATETIME,
    
    -- Password reset
    reset_token VARCHAR(255),
    reset_token_exp DATETIME,
    
    -- Timestamps
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_users_email (email),
    INDEX idx_users_role (role),
    INDEX idx_users_deleted_at (deleted_at)
);

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    is_revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_refresh_tokens_user_id (user_id),
    INDEX idx_refresh_tokens_token (token),
    INDEX idx_refresh_tokens_expires_at (expires_at)
);

-- Packages
CREATE TABLE IF NOT EXISTS packages (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type ENUM('basic', 'premium', 'complete') NOT NULL,
    price DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Stripe integration
    stripe_product_id VARCHAR(255) UNIQUE,
    stripe_price_id VARCHAR(255) UNIQUE,
    
    -- Package features and settings
    features TEXT, -- JSON array
    requires_timeslot BOOLEAN NOT NULL DEFAULT TRUE,
    manual_assignment BOOLEAN NOT NULL DEFAULT FALSE,
    consultation_time INT DEFAULT 60,
    has_free_pre_talk BOOLEAN NOT NULL DEFAULT FALSE,
    pre_talk_duration INT DEFAULT 15,
    
    -- Display settings
    sort_order INT DEFAULT 0,
    badge_text VARCHAR(100),
    badge_color VARCHAR(50) DEFAULT 'primary',
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_packages_type (type),
    INDEX idx_packages_is_active (is_active),
    INDEX idx_packages_deleted_at (deleted_at)
);

-- Addons
CREATE TABLE IF NOT EXISTS addons (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Stripe integration
    stripe_product_id VARCHAR(255) UNIQUE,
    stripe_price_id VARCHAR(255) UNIQUE,
    
    -- Display settings
    sort_order INT DEFAULT 0,
    category VARCHAR(100) DEFAULT 'general',
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_addons_is_active (is_active),
    INDEX idx_addons_category (category),
    INDEX idx_addons_deleted_at (deleted_at)
);

-- Package-Addon relationships
CREATE TABLE IF NOT EXISTS package_addons (
    package_id CHAR(36),
    addon_id CHAR(36),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    
    PRIMARY KEY (package_id, addon_id),
    FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE CASCADE,
    FOREIGN KEY (addon_id) REFERENCES addons(id) ON DELETE CASCADE
);

-- Timeslots
CREATE TABLE IF NOT EXISTS timeslots (
    id CHAR(36) PRIMARY KEY,
    berater_id CHAR(36) NOT NULL,
    
    -- Time details
    date DATE NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    duration INT NOT NULL,
    
    -- Availability
    is_available BOOLEAN NOT NULL DEFAULT TRUE,
    is_recurring BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Recurrence settings
    recurrence_pattern VARCHAR(50),
    recurrence_end DATETIME,
    
    -- Booking limits
    max_bookings INT NOT NULL DEFAULT 1,
    current_bookings INT NOT NULL DEFAULT 0,
    
    -- Metadata
    title VARCHAR(255),
    description TEXT,
    location VARCHAR(255),
    is_online BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (berater_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_timeslots_berater_id (berater_id),
    INDEX idx_timeslots_date (date),
    INDEX idx_timeslots_start_time (start_time),
    INDEX idx_timeslots_is_available (is_available),
    INDEX idx_timeslots_deleted_at (deleted_at)
);

-- Leads (extended)
CREATE TABLE IF NOT EXISTS leads (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    berater_id CHAR(36),
    
    -- Lead information
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status ENUM('neu', 'in_bearbeitung', 'rückfrage', 'abgeschlossen', 'storniert', 'zahlung_ausstehend') NOT NULL DEFAULT 'neu',
    priority ENUM('niedrig', 'mittel', 'hoch', 'dringend') NOT NULL DEFAULT 'mittel',
    
    -- Lead source and tracking
    source ENUM('website', 'booking', 'contact_form', 'referral', 'phone', 'email', 'social_media', 'manual') NOT NULL DEFAULT 'manual',
    source_details TEXT,
    referral_source VARCHAR(255),
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    
    -- Contact attempt tracking
    contact_attempts INT DEFAULT 0,
    last_contact_at DATETIME,
    next_follow_up_at DATETIME,
    next_follow_up_note TEXT,
    
    -- Qualification
    is_qualified BOOLEAN NOT NULL DEFAULT FALSE,
    qualification_notes TEXT,
    qualified_at DATETIME,
    
    -- Value estimation
    estimated_value DECIMAL(10,2) DEFAULT 0,
    estimated_close_date DATETIME,
    
    -- Communication preferences
    preferred_contact_method VARCHAR(50) DEFAULT 'email',
    preferred_contact_time VARCHAR(50),
    timezone VARCHAR(50) DEFAULT 'Europe/Berlin',
    
    -- Lead scoring
    lead_score INT DEFAULT 0,
    lead_score_reason TEXT,
    
    -- Conversion tracking
    converted_at DATETIME,
    conversion_value DECIMAL(10,2) DEFAULT 0,
    
    -- Elterngeld specific fields
    child_name VARCHAR(255),
    child_birth_date DATETIME,
    expected_amount DECIMAL(10,2),
    application_number VARCHAR(100) UNIQUE,
    
    -- Contact preferences
    preferred_contact VARCHAR(50) DEFAULT 'email',
    
    -- Timeline
    due_date DATETIME,
    completed_at DATETIME,
    
    -- Internal notes
    internal_notes TEXT,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (berater_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_leads_user_id (user_id),
    INDEX idx_leads_berater_id (berater_id),
    INDEX idx_leads_status (status),
    INDEX idx_leads_priority (priority),
    INDEX idx_leads_source (source),
    INDEX idx_leads_application_number (application_number),
    INDEX idx_leads_deleted_at (deleted_at)
);

-- Payments (existing)
CREATE TABLE IF NOT EXISTS payments (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    
    -- Payment information
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    status ENUM('pending', 'processing', 'succeeded', 'failed', 'canceled', 'refunded') NOT NULL DEFAULT 'pending',
    method ENUM('stripe', 'bank_transfer', 'cash') NOT NULL DEFAULT 'stripe',
    description TEXT,
    
    -- Stripe specific fields
    stripe_session_id VARCHAR(255) UNIQUE,
    stripe_payment_intent VARCHAR(255),
    stripe_customer_id VARCHAR(255),
    stripe_charge_id VARCHAR(255),
    
    -- Payment details
    payment_method_details TEXT,
    receipt_url VARCHAR(500),
    
    -- Billing information
    billing_name VARCHAR(255),
    billing_email VARCHAR(255),
    billing_address TEXT,
    
    -- Timestamps
    paid_at DATETIME,
    failed_at DATETIME,
    refunded_at DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    -- Failure information
    failure_code VARCHAR(100),
    failure_message TEXT,
    
    -- Refund information
    refund_amount DECIMAL(10,2) DEFAULT 0,
    refund_reason TEXT,
    
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_payments_lead_id (lead_id),
    INDEX idx_payments_user_id (user_id),
    INDEX idx_payments_status (status),
    INDEX idx_payments_stripe_session_id (stripe_session_id),
    INDEX idx_payments_deleted_at (deleted_at)
);

-- Bookings
CREATE TABLE IF NOT EXISTS bookings (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    package_id CHAR(36),
    berater_id CHAR(36),
    lead_id CHAR(36),
    payment_id CHAR(36),
    timeslot_id CHAR(36),
    
    -- Booking details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type ENUM('consultation', 'pre_talk', 'follow_up') NOT NULL DEFAULT 'consultation',
    status ENUM('pending', 'confirmed', 'completed', 'cancelled', 'no_show') NOT NULL DEFAULT 'pending',
    
    -- Timing
    scheduled_at DATETIME NOT NULL,
    duration INT NOT NULL DEFAULT 60,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    
    -- Contact information
    customer_name VARCHAR(255),
    customer_email VARCHAR(255),
    customer_phone VARCHAR(100),
    customer_address TEXT,
    customer_notes TEXT,
    
    -- Meeting details
    meeting_link VARCHAR(500),
    meeting_password VARCHAR(100),
    location VARCHAR(255),
    is_online BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Booking metadata
    booking_reference VARCHAR(100) UNIQUE,
    internal_notes TEXT,
    cancellation_note TEXT,
    
    -- Pricing
    total_amount DECIMAL(10,2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'EUR',
    
    -- Timestamps
    booked_at DATETIME NOT NULL,
    confirmed_at DATETIME,
    completed_at DATETIME,
    cancelled_at DATETIME,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (package_id) REFERENCES packages(id) ON DELETE SET NULL,
    FOREIGN KEY (berater_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE SET NULL,
    FOREIGN KEY (payment_id) REFERENCES payments(id) ON DELETE SET NULL,
    FOREIGN KEY (timeslot_id) REFERENCES timeslots(id) ON DELETE SET NULL,
    INDEX idx_bookings_user_id (user_id),
    INDEX idx_bookings_package_id (package_id),
    INDEX idx_bookings_berater_id (berater_id),
    INDEX idx_bookings_lead_id (lead_id),
    INDEX idx_bookings_timeslot_id (timeslot_id),
    INDEX idx_bookings_status (status),
    INDEX idx_bookings_scheduled_at (scheduled_at),
    INDEX idx_bookings_booking_reference (booking_reference),
    INDEX idx_bookings_deleted_at (deleted_at)
);

-- Booking-Addon relationships
CREATE TABLE IF NOT EXISTS booking_addons (
    booking_id CHAR(36),
    addon_id CHAR(36),
    price DECIMAL(10,2) NOT NULL,
    created_at DATETIME NOT NULL,
    
    PRIMARY KEY (booking_id, addon_id),
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE,
    FOREIGN KEY (addon_id) REFERENCES addons(id) ON DELETE CASCADE
);

-- Todos
CREATE TABLE IF NOT EXISTS todos (
    id CHAR(36) PRIMARY KEY,
    booking_id CHAR(36),
    lead_id CHAR(36),
    user_id CHAR(36) NOT NULL,
    created_by CHAR(36) NOT NULL,
    
    -- Todo details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timing
    due_date DATETIME,
    completed_at DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE SET NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE SET NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_todos_booking_id (booking_id),
    INDEX idx_todos_lead_id (lead_id),
    INDEX idx_todos_user_id (user_id),
    INDEX idx_todos_created_by (created_by),
    INDEX idx_todos_is_completed (is_completed),
    INDEX idx_todos_due_date (due_date),
    INDEX idx_todos_deleted_at (deleted_at)
);

-- Documents (existing)
CREATE TABLE IF NOT EXISTS documents (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    
    file_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    category VARCHAR(100) DEFAULT 'general',
    
    is_public BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    
    uploaded_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_documents_lead_id (lead_id),
    INDEX idx_documents_user_id (user_id),
    INDEX idx_documents_category (category),
    INDEX idx_documents_deleted_at (deleted_at)
);

-- Activities (existing)
CREATE TABLE IF NOT EXISTS activities (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    lead_id CHAR(36),
    
    type VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    metadata TEXT, -- JSON
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE SET NULL,
    INDEX idx_activities_user_id (user_id),
    INDEX idx_activities_lead_id (lead_id),
    INDEX idx_activities_type (type),
    INDEX idx_activities_deleted_at (deleted_at)
);

-- Comments
CREATE TABLE IF NOT EXISTS comments (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_comments_lead_id (lead_id),
    INDEX idx_comments_user_id (user_id),
    INDEX idx_comments_deleted_at (deleted_at)
);

-- Reminders
CREATE TABLE IF NOT EXISTS reminders (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    created_by CHAR(36) NOT NULL,
    
    title VARCHAR(255) NOT NULL,
    description TEXT,
    remind_at DATETIME NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_reminders_lead_id (lead_id),
    INDEX idx_reminders_user_id (user_id),
    INDEX idx_reminders_created_by (created_by),
    INDEX idx_reminders_remind_at (remind_at),
    INDEX idx_reminders_deleted_at (deleted_at)
);

-- Email threads
CREATE TABLE IF NOT EXISTS email_threads (
    id CHAR(36) PRIMARY KEY,
    lead_id CHAR(36) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    thread_id VARCHAR(255) UNIQUE,
    last_message_at DATETIME NOT NULL,
    message_count INT DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE,
    INDEX idx_email_threads_lead_id (lead_id),
    INDEX idx_email_threads_thread_id (thread_id),
    INDEX idx_email_threads_deleted_at (deleted_at)
);

-- Email messages
CREATE TABLE IF NOT EXISTS email_messages (
    id CHAR(36) PRIMARY KEY,
    thread_id CHAR(36) NOT NULL,
    message_id VARCHAR(255) UNIQUE,
    
    from_email VARCHAR(255) NOT NULL,
    to_email VARCHAR(255) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    body TEXT,
    is_html BOOLEAN NOT NULL DEFAULT FALSE,
    
    is_inbound BOOLEAN NOT NULL DEFAULT TRUE,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    
    sent_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    
    FOREIGN KEY (thread_id) REFERENCES email_threads(id) ON DELETE CASCADE,
    INDEX idx_email_messages_thread_id (thread_id),
    INDEX idx_email_messages_message_id (message_id),
    INDEX idx_email_messages_sent_at (sent_at)
);

-- Notifications
CREATE TABLE IF NOT EXISTS notifications (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    type ENUM('email', 'sms', 'in_app', 'push') NOT NULL,
    status ENUM('pending', 'sent', 'delivered', 'failed', 'retrying') NOT NULL DEFAULT 'pending',
    
    -- Content
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    data TEXT, -- JSON
    
    -- Template information
    template VARCHAR(100),
    template_data TEXT, -- JSON
    
    -- Recipients
    recipient VARCHAR(255) NOT NULL,
    cc_recipients TEXT,
    bcc_recipients TEXT,
    
    -- Delivery tracking
    sent_at DATETIME,
    delivered_at DATETIME,
    failed_at DATETIME,
    read_at DATETIME,
    
    -- Retry mechanism
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    next_retry_at DATETIME,
    
    -- Error tracking
    error_message TEXT,
    
    -- External IDs
    external_id VARCHAR(255),
    
    -- Priority and scheduling
    priority INT DEFAULT 0,
    schedule_at DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_notifications_user_id (user_id),
    INDEX idx_notifications_type (type),
    INDEX idx_notifications_status (status),
    INDEX idx_notifications_schedule_at (schedule_at),
    INDEX idx_notifications_deleted_at (deleted_at)
);

-- Email verifications
CREATE TABLE IF NOT EXISTS email_verifications (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL,
    
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at DATETIME,
    
    verification_attempts INT DEFAULT 0,
    last_attempt_at DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_email_verifications_user_id (user_id),
    INDEX idx_email_verifications_email (email),
    INDEX idx_email_verifications_token (token),
    INDEX idx_email_verifications_deleted_at (deleted_at)
);

-- Password resets
CREATE TABLE IF NOT EXISTS password_resets (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    email VARCHAR(255) NOT NULL,
    
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_password_resets_user_id (user_id),
    INDEX idx_password_resets_email (email),
    INDEX idx_password_resets_token (token),
    INDEX idx_password_resets_deleted_at (deleted_at)
);

-- Notification preferences
CREATE TABLE IF NOT EXISTS notification_preferences (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL UNIQUE,
    
    -- Email preferences
    email_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    email_booking_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    email_payment_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    email_marketing_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    email_todo_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    email_reminder_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- SMS preferences
    sms_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    sms_booking_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    sms_reminder_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- In-app preferences
    in_app_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    in_app_booking_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    in_app_todo_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Push preferences
    push_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    push_booking_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    push_reminder_notifications BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Timing preferences
    quiet_hours_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    quiet_hours_start TIME,
    quiet_hours_end TIME,
    timezone VARCHAR(50) DEFAULT 'Europe/Berlin',
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_notification_preferences_deleted_at (deleted_at)
);

-- Contact forms
CREATE TABLE IF NOT EXISTS contact_forms (
    id CHAR(36) PRIMARY KEY,
    
    -- Contact information
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(100),
    subject VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    
    -- Additional context
    source VARCHAR(100) DEFAULT 'website',
    url VARCHAR(500),
    user_agent TEXT,
    ip_address VARCHAR(45),
    
    -- UTM tracking
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    utm_term VARCHAR(255),
    utm_content VARCHAR(255),
    
    -- Processing status
    is_processed BOOLEAN NOT NULL DEFAULT FALSE,
    processed_at DATETIME,
    processed_by CHAR(36),
    lead_created BOOLEAN NOT NULL DEFAULT FALSE,
    lead_id CHAR(36),
    
    -- Response tracking
    is_replied BOOLEAN NOT NULL DEFAULT FALSE,
    replied_at DATETIME,
    replied_by CHAR(36),
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (processed_by) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (replied_by) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE SET NULL,
    INDEX idx_contact_forms_email (email),
    INDEX idx_contact_forms_source (source),
    INDEX idx_contact_forms_is_processed (is_processed),
    INDEX idx_contact_forms_deleted_at (deleted_at)
);

-- Permissions
CREATE TABLE IF NOT EXISTS permissions (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_permissions_name (name),
    INDEX idx_permissions_resource (resource),
    INDEX idx_permissions_action (action),
    INDEX idx_permissions_is_active (is_active),
    INDEX idx_permissions_deleted_at (deleted_at)
);

-- Roles
CREATE TABLE IF NOT EXISTS roles (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT DEFAULT 0,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_roles_name (name),
    INDEX idx_roles_is_active (is_active),
    INDEX idx_roles_is_default (is_default),
    INDEX idx_roles_deleted_at (deleted_at)
);

-- Role-Permission relationships
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id CHAR(36),
    permission_id CHAR(36),
    granted_at DATETIME NOT NULL,
    granted_by CHAR(36) NOT NULL,
    
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (granted_by) REFERENCES users(id) ON DELETE CASCADE
);

-- User-Role relationships
CREATE TABLE IF NOT EXISTS user_roles (
    user_id CHAR(36),
    role_id CHAR(36),
    assigned_at DATETIME NOT NULL,
    assigned_by CHAR(36) NOT NULL,
    expires_at DATETIME,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_by) REFERENCES users(id) ON DELETE CASCADE
);

-- User permissions (direct assignments)
CREATE TABLE IF NOT EXISTS user_permissions (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    permission_id CHAR(36) NOT NULL,
    is_granted BOOLEAN NOT NULL DEFAULT TRUE,
    granted_at DATETIME NOT NULL,
    granted_by CHAR(36) NOT NULL,
    expires_at DATETIME,
    reason TEXT,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE,
    FOREIGN KEY (granted_by) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_permissions_user_id (user_id),
    INDEX idx_user_permissions_permission_id (permission_id),
    INDEX idx_user_permissions_deleted_at (deleted_at)
);

-- Permission templates
CREATE TABLE IF NOT EXISTS permission_templates (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    permission_names TEXT NOT NULL,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    INDEX idx_permission_templates_name (name),
    INDEX idx_permission_templates_is_active (is_active),
    INDEX idx_permission_templates_deleted_at (deleted_at)
);

-- Jobs
CREATE TABLE IF NOT EXISTS jobs (
    id CHAR(36) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    short_description TEXT,
    
    -- Job details
    status ENUM('draft', 'published', 'paused', 'closed', 'archived') NOT NULL DEFAULT 'draft',
    type ENUM('full_time', 'part_time', 'contract', 'internship', 'freelance') NOT NULL,
    level ENUM('entry', 'junior', 'mid', 'senior', 'lead') NOT NULL,
    department VARCHAR(255),
    location VARCHAR(255) NOT NULL,
    work_location ENUM('remote', 'on_site', 'hybrid') NOT NULL DEFAULT 'on_site',
    is_remote BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Compensation
    salary_min DECIMAL(10,2),
    salary_max DECIMAL(10,2),
    salary_currency VARCHAR(3) DEFAULT 'EUR',
    salary_period VARCHAR(20) DEFAULT 'yearly',
    benefits_text TEXT,
    
    -- Requirements
    required_skills TEXT, -- JSON
    preferred_skills TEXT, -- JSON
    required_experience TEXT,
    education_required TEXT,
    language_requirements TEXT,
    
    -- Application settings
    application_deadline DATETIME,
    contact_email VARCHAR(255),
    application_url VARCHAR(500),
    allow_direct_apply BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- SEO and metadata
    meta_title VARCHAR(255),
    meta_description TEXT,
    tags TEXT, -- JSON
    
    -- Tracking
    view_count INT DEFAULT 0,
    application_count INT DEFAULT 0,
    
    -- Publication
    published_at DATETIME,
    expires_at DATETIME,
    
    -- Management
    created_by CHAR(36) NOT NULL,
    updated_by CHAR(36),
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (updated_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_jobs_slug (slug),
    INDEX idx_jobs_status (status),
    INDEX idx_jobs_type (type),
    INDEX idx_jobs_level (level),
    INDEX idx_jobs_location (location),
    INDEX idx_jobs_published_at (published_at),
    INDEX idx_jobs_deleted_at (deleted_at)
);

-- Job applications
CREATE TABLE IF NOT EXISTS job_applications (
    id CHAR(36) PRIMARY KEY,
    job_id CHAR(36) NOT NULL,
    
    -- Applicant information
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(100),
    location VARCHAR(255),
    
    -- Application details
    status ENUM('submitted', 'reviewing', 'screening', 'interview', 'offered', 'accepted', 'rejected', 'withdrawn') NOT NULL DEFAULT 'submitted',
    cover_letter TEXT,
    resume_url VARCHAR(500),
    portfolio_url VARCHAR(500),
    linkedin_url VARCHAR(500),
    github_url VARCHAR(500),
    website_url VARCHAR(500),
    
    -- Experience and skills
    years_experience INT DEFAULT 0,
    current_position VARCHAR(255),
    current_company VARCHAR(255),
    expected_salary DECIMAL(10,2),
    availability_date DATETIME,
    notice_period VARCHAR(100),
    
    -- Additional information
    motivation_text TEXT,
    questions TEXT,
    privacy_consent BOOLEAN NOT NULL DEFAULT FALSE,
    newsletter_consent BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Tracking and source
    source VARCHAR(100) DEFAULT 'website',
    source_details VARCHAR(255),
    referral_name VARCHAR(255),
    utm_source VARCHAR(255),
    utm_medium VARCHAR(255),
    utm_campaign VARCHAR(255),
    
    -- Review process
    reviewed_by CHAR(36),
    reviewed_at DATETIME,
    review_notes TEXT,
    rejection_note TEXT,
    
    -- Communication tracking
    last_contact_at DATETIME,
    next_follow_up_at DATETIME,
    interview_scheduled BOOLEAN NOT NULL DEFAULT FALSE,
    interview_date DATETIME,
    
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_job_applications_job_id (job_id),
    INDEX idx_job_applications_email (email),
    INDEX idx_job_applications_status (status),
    INDEX idx_job_applications_reviewed_by (reviewed_by),
    INDEX idx_job_applications_deleted_at (deleted_at)
);

-- Job application documents
CREATE TABLE IF NOT EXISTS job_application_documents (
    id CHAR(36) PRIMARY KEY,
    application_id CHAR(36) NOT NULL,
    
    file_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    file_type VARCHAR(100) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    document_type VARCHAR(100) NOT NULL,
    
    uploaded_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    deleted_at DATETIME,
    
    FOREIGN KEY (application_id) REFERENCES job_applications(id) ON DELETE CASCADE,
    INDEX idx_job_application_documents_application_id (application_id),
    INDEX idx_job_application_documents_document_type (document_type),
    INDEX idx_job_application_documents_deleted_at (deleted_at)
);

-- Job application activities
CREATE TABLE IF NOT EXISTS job_application_activities (
    id CHAR(36) PRIMARY KEY,
    application_id CHAR(36) NOT NULL,
    user_id CHAR(36),
    
    type VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    details TEXT, -- JSON
    
    old_value VARCHAR(255),
    new_value VARCHAR(255),
    
    created_at DATETIME NOT NULL,
    
    FOREIGN KEY (application_id) REFERENCES job_applications(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_job_application_activities_application_id (application_id),
    INDEX idx_job_application_activities_user_id (user_id),
    INDEX idx_job_application_activities_type (type)
);