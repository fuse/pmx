package openid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// Session is a Proxmox API session obtained via OpenID login.
type Session struct {
	Username            string `json:"username"`
	Ticket              string `json:"ticket"`
	CSRFPreventionToken string `json:"CSRFPreventionToken"`
}

// Client talks to the unauthenticated Proxmox OpenID endpoints.
type Client struct {
	Endpoint   string // e.g. https://pve.example.com
	HTTPClient *http.Client
}

func (c *Client) http() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: defaultTimeout}
}

func (c *Client) apiURL(path string) (string, error) {
	base := strings.TrimRight(c.Endpoint, "/")
	if base == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	return base + "/api2/json" + path, nil
}

type apiStringResponse struct {
	Data string `json:"data"`
}

type apiSessionResponse struct {
	Data Session `json:"data"`
}

// AuthURL requests the IdP authorization URL for realm + redirectURL.
func (c *Client) AuthURL(ctx context.Context, realm, redirectURL string) (string, error) {
	endpoint, err := c.apiURL("/access/openid/auth-url")
	if err != nil {
		return "", err
	}

	form := url.Values{}
	form.Set("realm", realm)
	form.Set("redirect-url", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("auth-url request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth-url: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed apiStringResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("auth-url: decode response: %w", err)
	}
	if parsed.Data == "" {
		return "", fmt.Errorf("auth-url: empty authorization URL in response")
	}
	return parsed.Data, nil
}

// Login exchanges the OpenID authorization code for a Proxmox ticket.
func (c *Client) Login(ctx context.Context, code, state, redirectURL string) (*Session, error) {
	endpoint, err := c.apiURL("/access/openid/login")
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("code", code)
	form.Set("state", state)
	form.Set("redirect-url", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http().Do(req)
	if err != nil {
		return nil, fmt.Errorf("openid login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openid login: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed apiSessionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("openid login: decode response: %w", err)
	}
	if parsed.Data.Ticket == "" || parsed.Data.CSRFPreventionToken == "" {
		return nil, fmt.Errorf("openid login: missing ticket or CSRF token in response")
	}
	return &parsed.Data, nil
}

// VerifyTicket checks whether ticket is still accepted by the Proxmox API.
func (c *Client) VerifyTicket(ctx context.Context, ticket string) error {
	endpoint, err := c.apiURL("/version")
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: ticket})

	resp, err := c.http().Do(req)
	if err != nil {
		return fmt.Errorf("verify ticket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("verify ticket: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
}
