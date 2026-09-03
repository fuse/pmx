package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fuse/pmx/internal/openid"
)

// DefaultTicketLifetime matches the Proxmox VE API ticket default (7200 s).
const DefaultTicketLifetime = 2 * time.Hour

// Stored is a persisted OpenID session for local reuse / curl tests.
type Stored struct {
	Endpoint            string    `json:"endpoint"`
	Realm               string    `json:"realm"`
	Username            string    `json:"username"`
	Ticket              string    `json:"ticket"`
	CSRFPreventionToken string    `json:"csrf_prevention_token"`
	ObtainedAt          time.Time `json:"obtained_at"`
	ExpiresAt           time.Time `json:"expires_at,omitempty"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pmx", "session.json"), nil
}

func Save(path string, endpoint, realm string, sess *openid.Session, ttl time.Duration) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	if ttl <= 0 {
		ttl = DefaultTicketLifetime
	}

	obtainedAt := time.Now().UTC()
	stored := Stored{
		Endpoint:            endpoint,
		Realm:               realm,
		Username:            sess.Username,
		Ticket:              sess.Ticket,
		CSRFPreventionToken: sess.CSRFPreventionToken,
		ObtainedAt:          obtainedAt,
		ExpiresAt:           obtainedAt.Add(ttl),
	}
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".session-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func Load(path string) (*Stored, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var stored Stored
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, err
	}
	if stored.Ticket == "" {
		return nil, fmt.Errorf("session file %s has no ticket", path)
	}
	stored.normalizeExpiry()
	return &stored, nil
}

func (s *Stored) normalizeExpiry() {
	if s.ExpiresAt.IsZero() && !s.ObtainedAt.IsZero() {
		s.ExpiresAt = s.ObtainedAt.Add(DefaultTicketLifetime)
	}
}

func (s *Stored) ExpiredAt(now time.Time) bool {
	s.normalizeExpiry()
	return !now.Before(s.ExpiresAt)
}

func (s *Stored) Remaining(now time.Time) time.Duration {
	s.normalizeExpiry()
	return s.ExpiresAt.Sub(now)
}

// Remove deletes the session file. Returns false when the file is already absent.
func Remove(path string) (bool, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return false, err
		}
	}
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
