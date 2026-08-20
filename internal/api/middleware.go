package api

import (
	"net/http"
	"strings"

	"github.com/relentlessworks/pollkit/internal/auth"
)

// authMiddleware extracts the bearer token and sets the workspace in context.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeError(w, r, http.StatusUnauthorized,
				"missing auth token",
				"call POST /auth/request with email to get an OTP, then POST /auth/verify to get a bearer token")
			return
		}
		t, ok := s.auth.ValidateToken(token)
		if !ok {
			writeError(w, r, http.StatusUnauthorized,
				"invalid or expired token",
				"call POST /auth/request with email to get a new OTP, then POST /auth/verify")
			return
		}
		r.Header.Set("X-Workspace", t.Workspace)
		r.Header.Set("X-Email", t.Email)
		next(w, r)
	}
}

// optionalAuth allows public access but sets workspace if token is present.
func (s *Server) optionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			t, ok := s.auth.ValidateToken(token)
			if ok {
				r.Header.Set("X-Workspace", t.Workspace)
				r.Header.Set("X-Email", t.Email)
			}
		}
		next(w, r)
	}
}

func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

func getWorkspace(r *http.Request) string {
	return r.Header.Get("X-Workspace")
}

func getEmail(r *http.Request) string {
	return r.Header.Get("X-Email")
}

// Ensure auth is used to avoid unused import
var _ = auth.New
