package store

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/relentlessworks/pollkit/internal/model"
)

// Store is a JSON file-backed data store.
type Store struct {
	mu   sync.RWMutex
	file string
	data *dbData
}

type dbData struct {
	Workspaces map[string]*model.Workspace `json:"workspaces"`
	Polls      map[string]*model.Poll      `json:"polls"`
	Votes      map[string]*model.Vote      `json:"votes"`
	Tokens     map[string]*model.Token     `json:"tokens"`
	OTPs       map[string]*model.OTP       `json:"otps"`
}

// New creates a new store backed by the given file path.
func New(file string) (*Store, error) {
	s := &Store{
		file: file,
		data: &dbData{
			Workspaces: make(map[string]*model.Workspace),
			Polls:      make(map[string]*model.Poll),
			Votes:      make(map[string]*model.Vote),
			Tokens:     make(map[string]*model.Token),
			OTPs:       make(map[string]*model.OTP),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	b, err := os.ReadFile(s.file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(b, s.data)
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.file, b, 0644)
}

// --- Workspace ---

func (s *Store) SaveWorkspace(w *model.Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Workspaces[w.Handle] = w
	return s.save()
}

func (s *Store) GetWorkspace(handle string) (*model.Workspace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w, ok := s.data.Workspaces[handle]
	return w, ok
}

// --- Poll ---

func (s *Store) SavePoll(p *model.Poll) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Polls[p.Handle] = p
	return s.save()
}

func (s *Store) GetPoll(handle string) (*model.Poll, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.data.Polls[handle]
	return p, ok
}

func (s *Store) ListPolls(workspace string) []*model.Poll {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*model.Poll
	for _, p := range s.data.Polls {
		if p.Workspace == workspace {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) DeletePoll(handle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Polls, handle)
	for vh, v := range s.data.Votes {
		if v.Poll == handle {
			delete(s.data.Votes, vh)
		}
	}
	return s.save()
}

// --- Vote ---

func (s *Store) SaveVote(v *model.Vote) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Votes[v.Handle] = v
	return s.save()
}

func (s *Store) GetVotes(pollHandle string) []*model.Vote {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*model.Vote
	for _, v := range s.data.Votes {
		if v.Poll == pollHandle {
			out = append(out, v)
		}
	}
	return out
}

func (s *Store) GetVoteByVoter(pollHandle, voter string) (*model.Vote, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.data.Votes {
		if v.Poll == pollHandle && v.Voter == voter {
			return v, true
		}
	}
	return nil, false
}

func (s *Store) DeleteVote(handle string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Votes, handle)
	return s.save()
}

// --- Token ---

func (s *Store) SaveToken(t *model.Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Tokens[t.Token] = t
	return s.save()
}

func (s *Store) GetToken(token string) (*model.Token, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.data.Tokens[token]
	return t, ok
}

// --- OTP ---

func (s *Store) SaveOTP(o *model.OTP) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.OTPs[o.Email] = o
	return s.save()
}

func (s *Store) GetOTP(email string) (*model.OTP, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.data.OTPs[email]
	return o, ok
}

func (s *Store) DeleteOTP(email string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.OTPs, email)
	s.save()
}

// CleanExpiredOTPs removes OTPs that have expired.
func (s *Store) CleanExpiredOTPs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for email, otp := range s.data.OTPs {
		if now.After(otp.ExpiresAt) {
			delete(s.data.OTPs, email)
		}
	}
	s.save()
}
