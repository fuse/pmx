package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fuse/pmx/internal/openid"
)

func TestStoredExpiry(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	stored := Stored{
		ObtainedAt: now,
		ExpiresAt:  now.Add(time.Hour),
	}

	if stored.ExpiredAt(now.Add(30 * time.Minute)) {
		t.Fatal("expected session still valid")
	}
	if !stored.ExpiredAt(now.Add(time.Hour)) {
		t.Fatal("expected session expired")
	}
	if got := stored.Remaining(now); got != time.Hour {
		t.Fatalf("remaining=%s", got)
	}
}

func TestStoredLegacyMissingExpiresAt(t *testing.T) {
	obtained := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	stored := Stored{ObtainedAt: obtained}
	stored.normalizeExpiry()
	want := obtained.Add(DefaultTicketLifetime)
	if !stored.ExpiresAt.Equal(want) {
		t.Fatalf("expires=%s want=%s", stored.ExpiresAt, want)
	}
}

func TestSaveSetsExpiresAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	sess := &openid.Session{
		Username:            "admin@openid",
		Ticket:              "ticket-value",
		CSRFPreventionToken: "csrf-value",
	}
	if err := Save(path, "https://pve.example.com", "myoidc", sess, 90*time.Minute); err != nil {
		t.Fatal(err)
	}

	stored, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Remaining(stored.ObtainedAt) != 90*time.Minute {
		t.Fatalf("remaining=%s", stored.Remaining(stored.ObtainedAt))
	}
}

func TestRemoveExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	sess := &openid.Session{Username: "u", Ticket: "t", CSRFPreventionToken: "c"}
	if err := Save(path, "https://pve.example.com", "realm", sess, time.Hour); err != nil {
		t.Fatal(err)
	}
	removed, err := Remove(path)
	if err != nil || !removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("stat=%v", err)
	}
}

func TestRemoveMissing(t *testing.T) {
	removed, err := Remove(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}
}
