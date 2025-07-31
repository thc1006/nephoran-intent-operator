package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// MFAManager handles multi-factor authentication
type MFAManager struct {
	config      *MFAConfig
	logger      *slog.Logger
	codes       map[string]*MFACode // temporary codes storage
	codesMutex  sync.RWMutex
	emailClient *EmailClient
	smsClient   *SMSClient
	metrics     *MFAMetrics
}

// MFAConfig holds MFA configuration
type MFAConfig struct {
	Enabled                bool          `json:"enabled"`
	RequiredForAdmin       bool          `json:"required_for_admin"`
	RequiredForOperator    bool          `json:"required_for_operator"`
	TOTPEnabled            bool          `json:"totp_enabled"`
	EmailEnabled           bool          `json:"email_enabled"`
	SMSEnabled             bool          `json:"sms_enabled"`
	BackupCodesEnabled     bool          `json:"backup_codes_enabled"`
	CodeLength             int           `json:"code_length"`
	CodeTTL                time.Duration `json:"code_ttl"`
	MaxAttempts            int           `json:"max_attempts"`
	AttemptWindow          time.Duration `json:"attempt_window"`
	CleanupInterval        time.Duration `json:"cleanup_interval"`
	IssuerName             string        `json:"issuer_name"`
	
	// Email configuration
	EmailConfig *EmailConfig `json:"email_config,omitempty"`
	
	// SMS configuration (integration with external service)
	SMSConfig *SMSConfig `json:"sms_config,omitempty"`
}

// EmailConfig holds email MFA configuration
type EmailConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromAddress  string `json:"from_address"`
	FromName     string `json:"from_name"`
	Subject      string `json:"subject"`
	Template     string `json:"template"`
}

// SMSConfig holds SMS MFA configuration
type SMSConfig struct {
	Provider   string `json:"provider"`   // twilio, aws-sns, etc.
	APIKey     string `json:"api_key"`
	APISecret  string `json:"api_secret"`
	FromNumber string `json:"from_number"`
	Template   string `json:"template"`
}

// MFACode represents a temporary MFA code
type MFACode struct {
	UserID      string    `json:"user_id"`
	Code        string    `json:"code"`
	Method      string    `json:"method"` // totp, email, sms
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Attempts    int       `json:"attempts"`
	Used        bool      `json:"used"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
}

// MFASetupData holds data for MFA setup
type MFASetupData struct {
	QRCodeURL    string   `json:"qr_code_url"`
	Secret       string   `json:"secret"`
	BackupCodes  []string `json:"backup_codes"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// MFAVerificationRequest represents an MFA verification request
type MFAVerificationRequest struct {
	UserID    string `json:"user_id"`
	Code      string `json:"code"`
	Method    string `json:"method"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// MFAVerificationResult represents the result of MFA verification
type MFAVerificationResult struct {
	Valid           bool      `json:"valid"`
	Method          string    `json:"method"`
	RemainingCodes  int       `json:"remaining_codes"`
	NextCodeAt      time.Time `json:"next_code_at"`
	AttemptsLeft    int       `json:"attempts_left"`
	BackupUsed      bool      `json:"backup_used"`
	Error           string    `json:"error,omitempty"`
}

// MFAMetrics tracks MFA performance and security metrics
type MFAMetrics struct {
	TotalVerifications    int64            `json:"total_verifications"`
	SuccessfulVerifications int64          `json:"successful_verifications"`
	FailedVerifications   int64            `json:"failed_verifications"`
	MethodStats           map[string]int64 `json:"method_stats"`
	AverageVerificationTime time.Duration  `json:"average_verification_time"`
	BruteForceAttempts    int64            `json:"brute_force_attempts"`
	BackupCodesUsed       int64            `json:"backup_codes_used"`
	LastUpdated           time.Time        `json:"last_updated"`
	mutex                 sync.RWMutex
}

// NewMFAManager creates a new MFA manager
func NewMFAManager(config *MFAConfig) (*MFAManager, error) {
	if config == nil {
		config = getDefaultMFAConfig()
	}

	manager := &MFAManager{
		config: config,
		logger: slog.Default().With("component", "mfa-manager"),
		codes:  make(map[string]*MFACode),
		metrics: &MFAMetrics{
			MethodStats: make(map[string]int64),
			LastUpdated: time.Now(),
		},
	}

	// Initialize email client
	if config.EmailEnabled && config.EmailConfig != nil {
		manager.emailClient = NewEmailClient(config.EmailConfig)
	}

	// Initialize SMS client
	if config.SMSEnabled && config.SMSConfig != nil {
		manager.smsClient = NewSMSClient(config.SMSConfig)
	}

	// Start cleanup routine
	go manager.startCleanupRoutine()

	return manager, nil
}

// getDefaultMFAConfig returns default MFA configuration
func getDefaultMFAConfig() *MFAConfig {
	return &MFAConfig{
		Enabled:                true,
		RequiredForAdmin:       true,
		RequiredForOperator:    false,
		TOTPEnabled:            true,
		EmailEnabled:           true,
		SMSEnabled:             false,
		BackupCodesEnabled:     true,
		CodeLength:             6,
		CodeTTL:                5 * time.Minute,
		MaxAttempts:            3,
		AttemptWindow:          15 * time.Minute,
		CleanupInterval:        time.Hour,
		IssuerName:             "Nephoran Intent Operator",
	}
}

// SetupTOTP sets up TOTP for a user
func (mfa *MFAManager) SetupTOTP(ctx context.Context, userID, userEmail string) (*MFASetupData, error) {
	if !mfa.config.TOTPEnabled {
		return nil, fmt.Errorf("TOTP is not enabled")
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfa.config.IssuerName,
		AccountName: userEmail,
		SecretSize:  32,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	setupData := &MFASetupData{
		QRCodeURL: key.URL(),
		Secret:    key.Secret(),
	}

	// Generate backup codes if enabled
	if mfa.config.BackupCodesEnabled {
		backupCodes, err := mfa.generateBackupCodes(10)
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup codes: %w", err)
		}
		setupData.BackupCodes = backupCodes

		// Generate recovery codes (longer, more secure)
		recoveryCodes, err := mfa.generateRecoveryCodes(5)
		if err != nil {
			return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
		}
		setupData.RecoveryCodes = recoveryCodes
	}

	mfa.logger.Info("TOTP setup initiated", "user_id", userID, "user_email", userEmail)
	return setupData, nil
}

// SendEmailCode sends an MFA code via email
func (mfa *MFAManager) SendEmailCode(ctx context.Context, userID, email, ipAddress, userAgent string) error {
	if !mfa.config.EmailEnabled || mfa.emailClient == nil {
		return fmt.Errorf("email MFA is not enabled")
	}

	// Generate random code
	code, err := mfa.generateNumericCode(mfa.config.CodeLength)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Store code
	mfaCode := &MFACode{
		UserID:    userID,
		Code:      code,
		Method:    "email",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(mfa.config.CodeTTL),
		Attempts:  0,
		Used:      false,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	codeKey := mfa.getCodeKey(userID, "email")
	mfa.codesMutex.Lock()
	mfa.codes[codeKey] = mfaCode
	mfa.codesMutex.Unlock()

	// Send email
	if err := mfa.emailClient.SendMFACode(ctx, email, code, mfa.config.CodeTTL); err != nil {
		return fmt.Errorf("failed to send email code: %w", err)
	}

	mfa.updateMethodStats("email")
	mfa.logger.Info("Email MFA code sent", "user_id", userID, "email", email)
	return nil
}

// SendSMSCode sends an MFA code via SMS
func (mfa *MFAManager) SendSMSCode(ctx context.Context, userID, phoneNumber, ipAddress, userAgent string) error {
	if !mfa.config.SMSEnabled || mfa.smsClient == nil {
		return fmt.Errorf("SMS MFA is not enabled")
	}

	// Generate random code
	code, err := mfa.generateNumericCode(mfa.config.CodeLength)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	// Store code
	mfaCode := &MFACode{
		UserID:    userID,
		Code:      code,
		Method:    "sms",
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(mfa.config.CodeTTL),
		Attempts:  0,
		Used:      false,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	codeKey := mfa.getCodeKey(userID, "sms")
	mfa.codesMutex.Lock()
	mfa.codes[codeKey] = mfaCode
	mfa.codesMutex.Unlock()

	// Send SMS
	if err := mfa.smsClient.SendMFACode(ctx, phoneNumber, code); err != nil {
		return fmt.Errorf("failed to send SMS code: %w", err)
	}

	mfa.updateMethodStats("sms")
	mfa.logger.Info("SMS MFA code sent", "user_id", userID, "phone", phoneNumber)
	return nil
}

// VerifyCode verifies an MFA code
func (mfa *MFAManager) VerifyCode(ctx context.Context, request *MFAVerificationRequest) (*MFAVerificationResult, error) {
	startTime := time.Now()
	defer func() {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.TotalVerifications++
			m.AverageVerificationTime = (m.AverageVerificationTime*time.Duration(m.TotalVerifications-1) + 
				time.Since(startTime)) / time.Duration(m.TotalVerifications)
		})
	}()

	result := &MFAVerificationResult{
		Valid:  false,
		Method: request.Method,
	}

	switch request.Method {
	case "totp":
		return mfa.verifyTOTP(ctx, request)
	case "email", "sms":
		return mfa.verifyTemporaryCode(ctx, request)
	case "backup":
		return mfa.verifyBackupCode(ctx, request)
	default:
		result.Error = "unsupported MFA method"
		return result, nil
	}
}

// verifyTOTP verifies a TOTP code
func (mfa *MFAManager) verifyTOTP(ctx context.Context, request *MFAVerificationRequest) (*MFAVerificationResult, error) {
	result := &MFAVerificationResult{
		Valid:  false,
		Method: "totp",
	}

	// In production, retrieve user's TOTP secret from secure storage
	// For now, we'll simulate validation
	userSecret, err := mfa.getUserTOTPSecret(request.UserID)
	if err != nil {
		result.Error = "TOTP not configured for user"
		return result, nil
	}

	valid := totp.Validate(request.Code, userSecret, time.Now())
	result.Valid = valid

	if valid {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.SuccessfulVerifications++
			m.MethodStats["totp"]++
		})
		mfa.logger.Info("TOTP verification successful", "user_id", request.UserID)
	} else {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.FailedVerifications++
		})
		mfa.logger.Warn("TOTP verification failed", "user_id", request.UserID)
		result.Error = "invalid TOTP code"
	}

	// Calculate next code time
	result.NextCodeAt = time.Now().Add(30 * time.Second) // TOTP codes refresh every 30 seconds

	return result, nil
}

// verifyTemporaryCode verifies email or SMS codes
func (mfa *MFAManager) verifyTemporaryCode(ctx context.Context, request *MFAVerificationRequest) (*MFAVerificationResult, error) {
	result := &MFAVerificationResult{
		Valid:  false,
		Method: request.Method,
	}

	codeKey := mfa.getCodeKey(request.UserID, request.Method)
	
	mfa.codesMutex.Lock()
	storedCode, exists := mfa.codes[codeKey]
	if !exists {
		mfa.codesMutex.Unlock()
		result.Error = "no code found or code expired"
		return result, nil
	}

	// Check if code is expired
	if time.Now().After(storedCode.ExpiresAt) {
		delete(mfa.codes, codeKey)
		mfa.codesMutex.Unlock()
		result.Error = "code expired"
		return result, nil
	}

	// Check if code is already used
	if storedCode.Used {
		mfa.codesMutex.Unlock()
		result.Error = "code already used"
		return result, nil
	}

	// Check attempt limits
	if storedCode.Attempts >= mfa.config.MaxAttempts {
		delete(mfa.codes, codeKey)
		mfa.codesMutex.Unlock()
		result.Error = "maximum attempts exceeded"
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.BruteForceAttempts++
		})
		return result, nil
	}

	// Increment attempts
	storedCode.Attempts++
	result.AttemptsLeft = mfa.config.MaxAttempts - storedCode.Attempts

	// Verify code
	if storedCode.Code == request.Code {
		storedCode.Used = true
		result.Valid = true
		
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.SuccessfulVerifications++
			m.MethodStats[request.Method]++
		})
		
		mfa.logger.Info("Temporary code verification successful", 
			"user_id", request.UserID, 
			"method", request.Method)
	} else {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.FailedVerifications++
		})
		
		mfa.logger.Warn("Temporary code verification failed", 
			"user_id", request.UserID, 
			"method", request.Method,
			"attempts", storedCode.Attempts)
		
		result.Error = "invalid code"
	}

	mfa.codesMutex.Unlock()
	return result, nil
}

// verifyBackupCode verifies a backup/recovery code
func (mfa *MFAManager) verifyBackupCode(ctx context.Context, request *MFAVerificationRequest) (*MFAVerificationResult, error) {
	result := &MFAVerificationResult{
		Valid:  false,
		Method: "backup",
	}

	// In production, retrieve and validate backup codes from secure storage
	// This would involve checking against stored hashed backup codes
	valid, remainingCodes := mfa.validateBackupCode(request.UserID, request.Code)
	
	result.Valid = valid
	result.RemainingCodes = remainingCodes
	result.BackupUsed = valid

	if valid {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.SuccessfulVerifications++
			m.BackupCodesUsed++
			m.MethodStats["backup"]++
		})
		
		mfa.logger.Warn("Backup code used", 
			"user_id", request.UserID,
			"remaining_codes", remainingCodes)
	} else {
		mfa.updateMetrics(func(m *MFAMetrics) {
			m.FailedVerifications++
		})
		
		result.Error = "invalid backup code"
	}

	return result, nil
}

// Helper methods

func (mfa *MFAManager) generateNumericCode(length int) (string, error) {
	max := 1
	for i := 0; i < length; i++ {
		max *= 10
	}
	
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	// Convert to number
	num := 0
	for _, b := range bytes {
		num = num*256 + int(b)
	}
	
	code := strconv.Itoa(num % max)
	
	// Pad with zeros if necessary
	for len(code) < length {
		code = "0" + code
	}
	
	return code, nil
}

func (mfa *MFAManager) generateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := mfa.generateAlphanumericCode(8)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func (mfa *MFAManager) generateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := mfa.generateAlphanumericCode(16)
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func (mfa *MFAManager) generateAlphanumericCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	
	return string(bytes), nil
}

func (mfa *MFAManager) getCodeKey(userID, method string) string {
	return userID + ":" + method
}

func (mfa *MFAManager) getUserTOTPSecret(userID string) (string, error) {
	// In production, retrieve from secure storage (database, vault, etc.)
	// For demo purposes, return a dummy secret
	return base32.StdEncoding.EncodeToString([]byte("dummy-secret-" + userID)), nil
}

func (mfa *MFAManager) validateBackupCode(userID, code string) (bool, int) {
	// In production, check against stored backup codes
	// Return whether code is valid and how many codes remain
	return true, 9 // Dummy response
}

func (mfa *MFAManager) updateMethodStats(method string) {
	mfa.metrics.mutex.Lock()
	defer mfa.metrics.mutex.Unlock()
	mfa.metrics.MethodStats[method]++
}

func (mfa *MFAManager) updateMetrics(updater func(*MFAMetrics)) {
	mfa.metrics.mutex.Lock()
	defer mfa.metrics.mutex.Unlock()
	updater(mfa.metrics)
	mfa.metrics.LastUpdated = time.Now()
}

func (mfa *MFAManager) startCleanupRoutine() {
	ticker := time.NewTicker(mfa.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		mfa.cleanupExpiredCodes()
	}
}

func (mfa *MFAManager) cleanupExpiredCodes() {
	mfa.codesMutex.Lock()
	defer mfa.codesMutex.Unlock()

	now := time.Now()
	for key, code := range mfa.codes {
		if now.After(code.ExpiresAt) || code.Used {
			delete(mfa.codes, key)
		}
	}
}

// GetMetrics returns current MFA metrics
func (mfa *MFAManager) GetMetrics() *MFAMetrics {
	mfa.metrics.mutex.RLock()
	defer mfa.metrics.mutex.RUnlock()
	
	metrics := *mfa.metrics
	return &metrics
}

// EmailClient handles email sending for MFA
type EmailClient struct {
	config *EmailConfig
}

func NewEmailClient(config *EmailConfig) *EmailClient {
	return &EmailClient{config: config}
}

func (ec *EmailClient) SendMFACode(ctx context.Context, email, code string, ttl time.Duration) error {
	subject := ec.config.Subject
	if subject == "" {
		subject = "Your Nephoran MFA Code"
	}

	body := fmt.Sprintf("Your verification code is: %s\n\nThis code will expire in %v.\n\nIf you didn't request this code, please contact your administrator.",
		code, ttl)

	if ec.config.Template != "" {
		body = strings.ReplaceAll(ec.config.Template, "{{code}}", code)
		body = strings.ReplaceAll(body, "{{ttl}}", ttl.String())
	}

	// Send email via SMTP
	auth := smtp.PlainAuth("", ec.config.SMTPUsername, ec.config.SMTPPassword, ec.config.SMTPHost)
	
	msg := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", email, subject, body)
	
	addr := fmt.Sprintf("%s:%d", ec.config.SMTPHost, ec.config.SMTPPort)
	return smtp.SendMail(addr, auth, ec.config.FromAddress, []string{email}, []byte(msg))
}

// SMSClient handles SMS sending for MFA
type SMSClient struct {
	config *SMSConfig
}

func NewSMSClient(config *SMSConfig) *SMSClient {
	return &SMSClient{config: config}
}

func (sc *SMSClient) SendMFACode(ctx context.Context, phoneNumber, code string) error {
	message := fmt.Sprintf("Your Nephoran verification code is: %s", code)
	
	if sc.config.Template != "" {
		message = strings.ReplaceAll(sc.config.Template, "{{code}}", code)
	}

	// In production, integrate with SMS providers like Twilio, AWS SNS, etc.
	// For now, just log the message
	slog.Info("SMS MFA code", "phone", phoneNumber, "code", code, "message", message)
	
	return nil
}