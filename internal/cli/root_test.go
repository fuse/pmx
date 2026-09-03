package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootVersion(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "pmx dev") {
		t.Fatalf("output=%q", out)
	}
}

func TestConfigPath(t *testing.T) {
	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"config", "path"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(buf.String()), filepath.Join(".config", "pmx", "config.yaml")) {
		t.Fatalf("output=%q", buf.String())
	}
}

func TestAuthLogoutMissingSession(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := NewRootCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"auth", "logout"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No saved session") {
		t.Fatalf("output=%q", buf.String())
	}
}
