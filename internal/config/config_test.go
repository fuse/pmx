package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeCallbackDefaults(t *testing.T) {
	cfg := &File{Endpoint: "https://pve.example.com/", Realm: "myoidc"}
	cfg.normalize()
	if cfg.RedirectURL != "http://127.0.0.1:8765/callback" {
		t.Fatalf("redirect=%q", cfg.RedirectURL)
	}
	if cfg.CallbackPort != DefaultCallbackPort {
		t.Fatalf("port=%d", cfg.CallbackPort)
	}
}

func TestLoadAndResolve(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte("endpoint: https://pve.example.com\nrealm: myoidc\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveAuth(cfg, "", "", "", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Realm != "myoidc" || resolved.RedirectURL != "http://127.0.0.1:8765/callback" {
		t.Fatalf("%+v", resolved)
	}
}

func TestResolveMissingEndpoint(t *testing.T) {
	_, err := ResolveAuth(&File{Realm: "x"}, "", "", "", "", 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveCallbackPortOverride(t *testing.T) {
	cfg := &File{Endpoint: "https://pve.example.com", Realm: "myoidc"}
	resolved, err := ResolveAuth(cfg, "", "", "", "", 9999)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.RedirectURL != "http://127.0.0.1:9999/callback" {
		t.Fatalf("%+v", resolved)
	}
}

func TestSessionTTLDurationDefault(t *testing.T) {
	d, err := (&File{}).SessionTTLDuration()
	if err != nil {
		t.Fatal(err)
	}
	if d != 2*time.Hour {
		t.Fatalf("ttl=%s", d)
	}
}

func TestSessionTTLDurationCustom(t *testing.T) {
	d, err := (&File{SessionTTL: "90m"}).SessionTTLDuration()
	if err != nil {
		t.Fatal(err)
	}
	if d != 90*time.Minute {
		t.Fatalf("ttl=%s", d)
	}
}

func TestSessionTTLDurationInvalid(t *testing.T) {
	if _, err := (&File{SessionTTL: "nope"}).SessionTTLDuration(); err == nil {
		t.Fatal("expected error")
	}
}
