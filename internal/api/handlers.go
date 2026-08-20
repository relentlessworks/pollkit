package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/relentlessworks/pollkit/internal/auth"
	"github.com/relentlessworks/pollkit/internal/model"
	"github.com/relentlessworks/pollkit/internal/store"
)

// Server holds all dependencies for the API.
type Server struct {
	store *store.Store
	auth  *auth.Auth
}

// New creates a new API server.
func New(s *store.Store, a *auth.Auth) *Server {
	return &Server{store: s, auth: a}
}

// Routes returns the HTTP mux with all routes registered.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/help", s.handleHelp)
	mux.HandleFunc("/.well-known/agent.md", s.handleHelp)
	mux.HandleFunc("/auth/request", s.handleAuthRequest)
	mux.HandleFunc("/auth/verify", s.handleAuthVerify)

	// Authenticated endpoints
	mux.HandleFunc("/polls", s.authMiddleware(s.handlePolls))
	mux.HandleFunc("/polls/", s.authMiddleware(s.handlePoll))

	// Public vote endpoint (optional auth)
	mux.HandleFunc("/vote", s.optionalAuth(s.handleVote))
	mux.HandleFunc("/results/", s.optionalAuth(s.handleResults))

	return mux
}

// --- Help ---

func (s *Server) handleHelp(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	help := `pollkit — agentic-first poll and survey service

AUTH:
  POST /auth/request  email=<email>          Request OTP (logged to stderr if no SMTP)
  POST /auth/verify   email=<email> code=<code>  Verify OTP, get bearer token

POLLS (requires Authorization: Bearer <token>):
  POST   /polls  question=<text> options=<comma-sep> [multiple=true] [public=true] [closes=<RFC3339>]
         → handle=poll_x1y2z question="..." options=3 multiple=false public=false
  GET    /polls                              List your polls
         → handle=poll_x1y2z question="..." votes=5 ...
  GET    /polls/<handle>                     Get poll details
  DELETE /polls/<handle>                     Delete poll and all votes

VOTING (public if poll is public, else requires token):
  POST /vote  poll=<handle> option=<option_id> [voter=<name>]
       → handle=vote_a1b2c poll=poll_x1y2z option=opt1 voter=alice
  DELETE /vote  poll=<handle> voter=<name>
       → (empty 200 on success)

RESULTS (public if poll is public, else requires token):
  GET /results/<poll_handle>
       → poll=poll_x1y2z total=5
         option_id=opt1 label="Yes" count=3 percent=60.0
         option_id=opt2 label="No" count=2 percent=40.0

FORMAT:
  Plain text by default. Add ?format=json or Accept: application/json for JSON.
  Errors: error: <message> | hint: <what to do next>
`
	w.Write([]byte(help))
}

// --- Auth ---

func (s *Server) handleAuthRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	email := r.FormValue("email")
	if email == "" {
		writeError(w, r, http.StatusBadRequest, "missing email", "provide email=<your@email.com>")
		return
	}
	wsHandle, err := s.auth.RequestOTP(email)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to send OTP", "check server logs")
		return
	}
	writeRecord(w, r, map[string]interface{}{
		"status":    "otp_sent",
		"email":     email,
		"workspace": wsHandle,
		"hint":      "check stderr for OTP code if no SMTP configured",
	})
}

func (s *Server) handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST")
		return
	}
	email := r.FormValue("email")
	code := r.FormValue("code")
	if email == "" || code == "" {
		writeError(w, r, http.StatusBadRequest, "missing email or code", "provide email=<your@email.com> code=<6-digit-code>")
		return
	}
	token, err := s.auth.VerifyOTP(email, code)
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, err.Error(), "request a new OTP via POST /auth/request")
		return
	}
	writeRecord(w, r, map[string]interface{}{
		"token":     token.Token,
		"workspace": token.Workspace,
		"email":     token.Email,
	})
}

// --- Polls ---

func (s *Server) handlePolls(w http.ResponseWriter, r *http.Request) {
	ws := getWorkspace(r)
	switch r.Method {
	case http.MethodPost:
		s.createPoll(w, r, ws)
	case http.MethodGet:
		s.listPolls(w, r, ws)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET or POST")
	}
}

func (s *Server) createPoll(w http.ResponseWriter, r *http.Request, ws string) {
	question := r.FormValue("question")
	if question == "" {
		writeError(w, r, http.StatusBadRequest, "missing question", "provide question=<your poll question>")
		return
	}
	optsStr := r.FormValue("options")
	if optsStr == "" {
		writeError(w, r, http.StatusBadRequest, "missing options", "provide options=<comma-separated choices>")
		return
	}
	optLabels := strings.Split(optsStr, ",")
	if len(optLabels) < 2 {
		writeError(w, r, http.StatusBadRequest, "need at least 2 options", "provide options=Yes,No,Maybe")
		return
	}

	var options []model.Option
	for i, label := range optLabels {
		label = strings.TrimSpace(label)
		options = append(options, model.Option{
			ID:    fmt.Sprintf("opt%d", i+1),
			Label: label,
		})
	}

	multiple := r.FormValue("multiple") == "true"
	public := r.FormValue("public") == "true"

	var closesAt *time.Time
	if closes := r.FormValue("closes"); closes != "" {
		t, err := time.Parse(time.RFC3339, closes)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid closes date", "use RFC3339 format like 2026-12-31T23:59:59Z")
			return
		}
		closesAt = &t
	}

	poll := &model.Poll{
		Handle:    model.GenPollHandle(),
		Workspace: ws,
		Question:  question,
		Options:   options,
		Multiple:  multiple,
		Public:    public,
		CreatedBy: getEmail(r),
		CreatedAt: time.Now(),
		ClosesAt:  closesAt,
	}

	if err := s.store.SavePoll(poll); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to save poll", "try again")
		return
	}

	fields := map[string]interface{}{
		"handle":   poll.Handle,
		"question": poll.Question,
		"options":  len(poll.Options),
		"multiple": poll.Multiple,
		"public":   poll.Public,
	}
	for _, opt := range poll.Options {
		fields[opt.ID] = opt.Label
	}
	writeRecord(w, r, fields)
}

func (s *Server) listPolls(w http.ResponseWriter, r *http.Request, ws string) {
	polls := s.store.ListPolls(ws)
	var records []map[string]interface{}
	for _, p := range polls {
		votes := s.store.GetVotes(p.Handle)
		records = append(records, map[string]interface{}{
			"handle":   p.Handle,
			"question": p.Question,
			"options":  len(p.Options),
			"votes":    len(votes),
			"public":   p.Public,
		})
	}
	if len(records) == 0 {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("no polls found\n"))
		return
	}
	writeRecords(w, r, records)
}

func (s *Server) handlePoll(w http.ResponseWriter, r *http.Request) {
	handle := strings.TrimPrefix(r.URL.Path, "/polls/")
	if handle == "" {
		writeError(w, r, http.StatusBadRequest, "missing poll handle", "use /polls/<handle>")
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.getPoll(w, r, handle)
	case http.MethodDelete:
		s.deletePoll(w, r, handle)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use GET or DELETE")
	}
}

func (s *Server) getPoll(w http.ResponseWriter, r *http.Request, handle string) {
	p, ok := s.store.GetPoll(handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "poll not found", "check the handle or call GET /polls to list your polls")
		return
	}
	ws := getWorkspace(r)
	if !p.Public && p.Workspace != ws {
		writeError(w, r, http.StatusForbidden, "poll is private", "provide a valid bearer token for the poll's workspace")
		return
	}
	votes := s.store.GetVotes(handle)
	fields := map[string]interface{}{
		"handle":   p.Handle,
		"question": p.Question,
		"options":  len(p.Options),
		"votes":    len(votes),
		"multiple": p.Multiple,
		"public":   p.Public,
	}
	for _, opt := range p.Options {
		fields[opt.ID] = opt.Label
	}
	if p.ClosesAt != nil {
		fields["closes_at"] = p.ClosesAt.Format(time.RFC3339)
	}
	writeRecord(w, r, fields)
}

func (s *Server) deletePoll(w http.ResponseWriter, r *http.Request, handle string) {
	ws := getWorkspace(r)
	p, ok := s.store.GetPoll(handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "poll not found", "check the handle or call GET /polls to list your polls")
		return
	}
	if p.Workspace != ws {
		writeError(w, r, http.StatusForbidden, "not your poll", "only the poll owner can delete it")
		return
	}
	if err := s.store.DeletePoll(handle); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to delete poll", "try again")
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("deleted poll " + handle + "\n"))
}

// --- Vote ---

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.castVote(w, r)
	case http.MethodDelete:
		s.removeVote(w, r)
	default:
		writeError(w, r, http.StatusMethodNotAllowed, "method not allowed", "use POST or DELETE")
	}
}

func (s *Server) castVote(w http.ResponseWriter, r *http.Request) {
	pollHandle := r.FormValue("poll")
	optionID := r.FormValue("option")
	voter := r.FormValue("voter")
	if voter == "" {
		voter = "anonymous"
	}
	if pollHandle == "" || optionID == "" {
		writeError(w, r, http.StatusBadRequest, "missing poll or option", "provide poll=<handle> option=<option_id> [voter=<name>]")
		return
	}

	p, ok := s.store.GetPoll(pollHandle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "poll not found", "check the poll handle")
		return
	}

	ws := getWorkspace(r)
	if !p.Public && p.Workspace != ws {
		writeError(w, r, http.StatusForbidden, "poll is private", "provide a valid bearer token for the poll's workspace")
		return
	}

	if p.ClosesAt != nil && time.Now().After(*p.ClosesAt) {
		writeError(w, r, http.StatusForbidden, "poll is closed", "voting ended at "+p.ClosesAt.Format(time.RFC3339))
		return
	}

	validOption := false
	for _, opt := range p.Options {
		if opt.ID == optionID {
			validOption = true
			break
		}
	}
	if !validOption {
		writeError(w, r, http.StatusBadRequest, "invalid option", "use one of: "+optionIDs(p.Options))
		return
	}

	if !p.Multiple {
		existing, found := s.store.GetVoteByVoter(pollHandle, voter)
		if found {
			existing.OptionID = optionID
			existing.CreatedAt = time.Now()
			if err := s.store.SaveVote(existing); err != nil {
				writeError(w, r, http.StatusInternalServerError, "failed to update vote", "try again")
				return
			}
			writeRecord(w, r, map[string]interface{}{
				"handle":  existing.Handle,
				"poll":    pollHandle,
				"option":  optionID,
				"voter":   voter,
				"updated": true,
			})
			return
		}
	}

	vote := &model.Vote{
		Handle:    model.GenVoteHandle(),
		Poll:      pollHandle,
		OptionID:  optionID,
		Voter:     voter,
		CreatedAt: time.Now(),
	}
	if err := s.store.SaveVote(vote); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to save vote", "try again")
		return
	}
	writeRecord(w, r, map[string]interface{}{
		"handle": vote.Handle,
		"poll":   pollHandle,
		"option": optionID,
		"voter":  voter,
	})
}

func (s *Server) removeVote(w http.ResponseWriter, r *http.Request) {
	pollHandle := r.FormValue("poll")
	voter := r.FormValue("voter")
	if pollHandle == "" || voter == "" {
		writeError(w, r, http.StatusBadRequest, "missing poll or voter", "provide poll=<handle> voter=<name>")
		return
	}
	vote, ok := s.store.GetVoteByVoter(pollHandle, voter)
	if !ok {
		writeError(w, r, http.StatusNotFound, "no vote found", "check the poll handle and voter name")
		return
	}
	if err := s.store.DeleteVote(vote.Handle); err != nil {
		writeError(w, r, http.StatusInternalServerError, "failed to delete vote", "try again")
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("deleted vote for " + voter + " on " + pollHandle + "\n"))
}

// --- Results ---

func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	handle := strings.TrimPrefix(r.URL.Path, "/results/")
	if handle == "" {
		writeError(w, r, http.StatusBadRequest, "missing poll handle", "use /results/<poll_handle>")
		return
	}
	p, ok := s.store.GetPoll(handle)
	if !ok {
		writeError(w, r, http.StatusNotFound, "poll not found", "check the poll handle")
		return
	}
	ws := getWorkspace(r)
	if !p.Public && p.Workspace != ws {
		writeError(w, r, http.StatusForbidden, "poll is private", "provide a valid bearer token for the poll's workspace")
		return
	}
	votes := s.store.GetVotes(handle)
	total := len(votes)

	counts := make(map[string]int)
	for _, v := range votes {
		counts[v.OptionID]++
	}

	var records []map[string]interface{}
	for _, opt := range p.Options {
		count := counts[opt.ID]
		var pct float64
		if total > 0 {
			pct = float64(count) / float64(total) * 100
		}
		records = append(records, map[string]interface{}{
			"option_id": opt.ID,
			"label":     opt.Label,
			"count":     count,
			"percent":   pct,
		})
	}

	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		out := map[string]interface{}{
			"poll":    handle,
			"total":   total,
			"results": records,
		}
		json.NewEncoder(w).Encode(out)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("poll=%s total=%s\n", handle, strconv.Itoa(total))))
	for _, rec := range records {
		var parts []string
		for k, v := range rec {
			parts = append(parts, k+"="+fmtVal(v))
		}
		w.Write([]byte(strings.Join(parts, " ") + "\n"))
	}
}

func optionIDs(opts []model.Option) string {
	var ids []string
	for _, o := range opts {
		ids = append(ids, o.ID)
	}
	return strings.Join(ids, ",")
}
