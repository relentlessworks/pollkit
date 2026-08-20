package model

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Poll represents a poll with multiple options.
type Poll struct {
	Handle    string    `json:"handle"`
	Workspace string    `json:"workspace"`
	Question  string    `json:"question"`
	Options   []Option  `json:"options"`
	Multiple  bool      `json:"multiple"`
	Public    bool      `json:"public"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	ClosesAt  *time.Time `json:"closes_at,omitempty"`
}

// Option is a single choice in a poll.
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Vote represents a single vote cast on a poll.
type Vote struct {
	Handle    string    `json:"handle"`
	Poll      string    `json:"poll"`
	OptionID  string    `json:"option_id"`
	Voter     string    `json:"voter"`
	CreatedAt time.Time `json:"created_at"`
}

// Result is the aggregated result for a poll option.
type Result struct {
	OptionID string  `json:"option_id"`
	Label    string  `json:"label"`
	Count    int     `json:"count"`
	Percent  float64 `json:"percent"`
}

// Workspace holds tenant data.
type Workspace struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	Plan   string `json:"plan"`
}

// Token is a bearer auth token.
type Token struct {
	Token     string    `json:"token"`
	Workspace string    `json:"workspace"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// OTP is a one-time password for auth.
type OTP struct {
	Code      string    `json:"code"`
	Email     string    `json:"email"`
	Workspace string    `json:"workspace"`
	ExpiresAt time.Time `json:"expires_at"`
}

// genHandle generates a short handle like poll_k7m2q.
func genHandle(prefix string) string {
	b := make([]byte, 5)
	rand.Read(b)
	return prefix + "_" + base32(b)
}

func base32(b []byte) string {
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 5 {
		s = s[:5]
	}
	return s
}

// GenPollHandle generates a poll handle.
func GenPollHandle() string { return genHandle("poll") }

// GenVoteHandle generates a vote handle.
func GenVoteHandle() string { return genHandle("vote") }

// GenWorkspaceHandle generates a workspace handle.
func GenWorkspaceHandle() string { return genHandle("ws") }

// GenToken generates a bearer token.
func GenToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenOTPCode generates a 6-digit OTP code.
func GenOTPCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	code := uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
	return zeroPad(int(code%1000000), 6)
}

func zeroPad(n, width int) string {
	s := ""
	v := n
	for i := 0; i < width; i++ {
		s = string(rune('0'+v%10)) + s
		v /= 10
	}
	return s
}
