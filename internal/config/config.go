package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultCallbackPort = 8765

// File is the on-disk YAML configuration (generic, no vendor defaults).
type File struct {
	// Endpoint is the Proxmox base URL (scheme + host, no /api2/json suffix).
	Endpoint string `yaml:"endpoint"`
	// Realm is the OpenID realm id configured on the Proxmox cluster.
	Realm string `yaml:"realm"`
	// RedirectURL is the OAuth redirect URI registered with the IdP.
	// Defaults to http://127.0.0.1:<callback_port>/callback.
	RedirectURL string `yaml:"redirect_url"`
	// CallbackPort is the loopback port for the local redirect listener (default 8765).
	CallbackPort int `yaml:"callback_port"`
	// SessionFile overrides the default session path (~/.config/pmx/session.json).
	SessionFile string `yaml:"session_file,omitempty"`
	// SessionTTL is the expected Proxmox ticket lifetime (default 2h).
	SessionTTL string `yaml:"session_ttl,omitempty"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "pmx", "config.yaml"), nil
}

// Load reads a YAML config file. Missing file returns empty File and nil error
// when path is the default path; an explicit path that does not exist is an error.
func Load(path string) (*File, error) {
	defaultPath, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	if path == "" {
		path = defaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && path == defaultPath {
			return &File{}, nil
		}
		return nil, err
	}

	var cfg File
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	cfg.normalize()
	return &cfg, nil
}

func (c *File) normalize() {
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	c.Realm = strings.TrimSpace(c.Realm)
	c.RedirectURL = strings.TrimSpace(c.RedirectURL)
	c.SessionFile = strings.TrimSpace(c.SessionFile)
	c.SessionTTL = strings.TrimSpace(c.SessionTTL)
	c.Endpoint = strings.TrimRight(c.Endpoint, "/")

	if c.CallbackPort <= 0 {
		c.CallbackPort = DefaultCallbackPort
	}
	if c.RedirectURL == "" {
		c.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d/callback", c.CallbackPort)
	}
}

// ResolveAuth merges CLI flag values (non-empty) over the file config.
func ResolveAuth(cfg *File, endpoint, realm, redirectURL, sessionFile string, callbackPort int) (File, error) {
	if cfg == nil {
		cfg = &File{}
	}
	out := *cfg
	if endpoint != "" {
		out.Endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	}
	if realm != "" {
		out.Realm = strings.TrimSpace(realm)
	}
	if redirectURL != "" {
		out.RedirectURL = strings.TrimSpace(redirectURL)
	}
	if sessionFile != "" {
		out.SessionFile = strings.TrimSpace(sessionFile)
	}
	if callbackPort > 0 {
		out.CallbackPort = callbackPort
		if redirectURL == "" {
			out.RedirectURL = ""
		}
	}

	out.normalize()

	if out.Endpoint == "" {
		return out, fmt.Errorf("endpoint is required (set it in the config file or pass --endpoint)")
	}
	if out.Realm == "" {
		return out, fmt.Errorf("realm is required (set it in the config file or pass --realm)")
	}
	if out.RedirectURL == "" {
		return out, fmt.Errorf("redirect_url is required (set it in the config file or pass --redirect-url)")
	}
	return out, nil
}

// SessionTTLDuration parses session_ttl or returns 2h when unset.
func (c *File) SessionTTLDuration() (time.Duration, error) {
	if c == nil || c.SessionTTL == "" {
		return 2 * time.Hour, nil
	}
	d, err := time.ParseDuration(c.SessionTTL)
	if err != nil {
		return 0, fmt.Errorf("invalid session_ttl %q: %w", c.SessionTTL, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("session_ttl must be positive, got %q", c.SessionTTL)
	}
	return d, nil
}
