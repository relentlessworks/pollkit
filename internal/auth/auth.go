package auth

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/relentlessworks/pollkit/internal/model"
	"github.com/relentlessworks/pollkit/internal/store"
)

// Auth handles OTP-based authentication.
type Auth struct {
	store *store.Store
	smtp  string
}

// New creates a new auth handler.
func New(s *store.Store, smtpURL string) *Auth {
	return &Auth{store: s, smtp: smtpURL}
}

// RequestOTP generates an OTP and sends it via email (or logs to stderr).
func (a *Auth) RequestOTP(email string) (string, error) {
	// Auto-create workspace if needed
	wsHandle := model.GenWorkspaceHandle()
	ws := &model.Workspace{
		Handle: wsHandle,
		Name:   email,
		Plan:   "free",
	}
	if err := a.store.SaveWorkspace(ws); err != nil {
		return "", err
	}

	code := model.GenOTPCode()
	otp := &model.OTP{
		Code:      code,
		Email:     email,
		Workspace: wsHandle,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := a.store.SaveOTP(otp); err != nil {
		return "", err
	}

	if a.smtp == "" {
		log.Printf("[OTP] %s: code=%s workspace=%s", email, code, wsHandle)
	} else {
		if err := a.sendEmail(email, code); err != nil {
			return "", fmt.Errorf("failed to send OTP email: %w", err)
		}
	}

	return wsHandle, nil
}

// VerifyOTP validates an OTP and returns a token.
func (a *Auth) VerifyOTP(email, code string) (*model.Token, error) {
	otp, ok := a.store.GetOTP(email)
	if !ok {
		return nil, fmt.Errorf("no OTP found for %s", email)
	}
	if time.Now().After(otp.ExpiresAt) {
		a.store.DeleteOTP(email)
		return nil, fmt.Errorf("OTP expired")
	}
	if otp.Code != code {
		return nil, fmt.Errorf("invalid OTP code")
	}

	token := &model.Token{
		Token:     model.GenToken(),
		Workspace: otp.Workspace,
		Email:     email,
		CreatedAt: time.Now(),
	}
	if err := a.store.SaveToken(token); err != nil {
		return nil, err
	}
	a.store.DeleteOTP(email)
	return token, nil
}

// ValidateToken checks a bearer token and returns the associated workspace.
func (a *Auth) ValidateToken(token string) (*model.Token, bool) {
	t, ok := a.store.GetToken(token)
	return t, ok
}

func (a *Auth) sendEmail(email, code string) error {
	parts := strings.Split(a.smtp, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid SMTP URL format")
	}
	host := parts[0]
	port := parts[1]
	addr := host + ":" + port
	msg := fmt.Sprintf("Subject: Your PollKit OTP\r\n\r\nYour verification code is: %s\r\n", code)
	return smtp.SendMail(addr, nil, "noreply@pollkit.local", []string{email}, []byte(msg))
}
