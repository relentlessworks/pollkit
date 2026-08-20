package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/relentlessworks/pollkit/internal/auth"
	"github.com/relentlessworks/pollkit/internal/store"
)

func setupTestServer(t *testing.T) (*Server, string, string) {
	t.Helper()
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	a := auth.New(s, "")
	srv := New(s, a)

	_, err = a.RequestOTP("test@example.com")
	if err != nil {
		t.Fatalf("failed to request OTP: %v", err)
	}
	otp, ok := s.GetOTP("test@example.com")
	if !ok {
		t.Fatalf("OTP not found")
	}
	token, err := a.VerifyOTP("test@example.com", otp.Code)
	if err != nil {
		t.Fatalf("failed to verify OTP: %v", err)
	}
	return srv, token.Token, token.Workspace
}

func TestHelp(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	mux := srv.Routes()

	req := httptest.NewRequest("GET", "/help", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "pollkit") {
		t.Errorf("help should contain 'pollkit', got: %s", body)
	}
}

func TestCreatePoll(t *testing.T) {
	srv, token, _ := setupTestServer(t)
	mux := srv.Routes()

	form := url.Values{}
	form.Set("question", "Best language?")
	form.Set("options", "Go,Python,Rust")
	form.Set("public", "true")

	req := httptest.NewRequest("POST", "/polls", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "handle=poll_") {
		t.Errorf("expected handle in response, got: %s", body)
	}
	if !strings.Contains(body, "opt1=Go") {
		t.Errorf("expected opt1=Go in response, got: %s", body)
	}
}

func TestVoteAndResults(t *testing.T) {
	srv, token, ws := setupTestServer(t)
	mux := srv.Routes()

	// Create poll
	form := url.Values{}
	form.Set("question", "Best language?")
	form.Set("options", "Go,Python")
	form.Set("public", "true")

	req := httptest.NewRequest("POST", "/polls", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	polls := srv.store.ListPolls(ws)
	if len(polls) == 0 {
		t.Fatal("no polls found")
	}
	pollHandle := polls[0].Handle

	// Vote
	voteForm := url.Values{}
	voteForm.Set("poll", pollHandle)
	voteForm.Set("option", "opt1")
	voteForm.Set("voter", "alice")

	req = httptest.NewRequest("POST", "/vote", strings.NewReader(voteForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Results
	req = httptest.NewRequest("GET", "/results/"+pollHandle, nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "total=1") {
		t.Errorf("expected total=1, got: %s", body)
	}
}

func TestJSONResults(t *testing.T) {
	srv, token, ws := setupTestServer(t)
	mux := srv.Routes()

	// Create poll
	form := url.Values{}
	form.Set("question", "Best language?")
	form.Set("options", "Go,Python")
	form.Set("public", "true")

	req := httptest.NewRequest("POST", "/polls", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	polls := srv.store.ListPolls(ws)
	pollHandle := polls[0].Handle

	// Vote
	voteForm := url.Values{}
	voteForm.Set("poll", pollHandle)
	voteForm.Set("option", "opt1")
	voteForm.Set("voter", "alice")

	req = httptest.NewRequest("POST", "/vote", strings.NewReader(voteForm.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	// JSON Results
	req = httptest.NewRequest("GET", "/results/"+pollHandle, nil)
	req.Header.Set("Accept", "application/json")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result["total"].(float64) != 1 {
		t.Errorf("expected total=1, got %v", result["total"])
	}
}

func TestAuthRequired(t *testing.T) {
	srv, _, _ := setupTestServer(t)
	mux := srv.Routes()

	req := httptest.NewRequest("GET", "/polls", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "hint:") {
		t.Errorf("error should contain hint, got: %s", body)
	}
}

func TestDeletePoll(t *testing.T) {
	srv, token, ws := setupTestServer(t)
	mux := srv.Routes()

	// Create poll
	form := url.Values{}
	form.Set("question", "Delete me?")
	form.Set("options", "Yes,No")

	req := httptest.NewRequest("POST", "/polls", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	polls := srv.store.ListPolls(ws)
	pollHandle := polls[0].Handle

	// Delete
	req = httptest.NewRequest("DELETE", "/polls/"+pollHandle, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "deleted") {
		t.Errorf("expected 'deleted' in response, got: %s", w.Body.String())
	}

	// Verify gone
	_, ok := srv.store.GetPoll(pollHandle)
	if ok {
		t.Error("poll should be deleted")
	}
}
