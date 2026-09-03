package openid

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const defaultCallbackHTML = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>pmx login</title></head>
<body>
  <h1>Authentication complete</h1>
  <p>You can close this window and return to the terminal.</p>
</body>
</html>`

// CallbackResult is the authorization code captured by the local redirect listener.
type CallbackResult struct {
	Code  string
	State string
}

// ListenRedirect starts a loopback HTTP server and waits for the IdP redirect.
// redirectURL must be of the form http://127.0.0.1:<port>/callback (or localhost).
func ListenRedirect(ctx context.Context, redirectURL string) (redirectURI string, wait func() (CallbackResult, error), cleanup func() error, err error) {
	u, err := url.Parse(redirectURL)
	if err != nil {
		return "", nil, nil, fmt.Errorf("parse redirect URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", nil, nil, fmt.Errorf("callback redirect URL must be http(s), got %q", u.Scheme)
	}
	host := u.Hostname()
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return "", nil, nil, fmt.Errorf("callback redirect host must be loopback, got %q", host)
	}
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	path := u.Path
	if path == "" {
		path = "/"
	}

	addr := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, nil, fmt.Errorf("listen on %s: %w", addr, err)
	}

	// Prefer the address we actually bound (handles :0 if ever used).
	bound := ln.Addr().(*net.TCPAddr)
	actualRedirect := fmt.Sprintf("%s://%s%s", u.Scheme, net.JoinHostPort(host, strconv.Itoa(bound.Port)), path)

	resultCh := make(chan CallbackResult, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			desc := q.Get("error_description")
			http.Error(w, fmt.Sprintf("IdP error: %s (%s)", errMsg, desc), http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("IdP error: %s: %s", errMsg, desc):
			default:
			}
			return
		}
		code := q.Get("code")
		state := q.Get("state")
		if code == "" || state == "" {
			http.Error(w, "missing code or state", http.StatusBadRequest)
			select {
			case errCh <- fmt.Errorf("callback missing code or state"):
			default:
			}
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(defaultCallbackHTML))
		select {
		case resultCh <- CallbackResult{Code: code, State: state}:
		default:
		}
	})

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if serveErr := srv.Serve(ln); serveErr != nil && serveErr != http.ErrServerClosed {
			select {
			case errCh <- serveErr:
			default:
			}
		}
	}()

	wait = func() (CallbackResult, error) {
		select {
		case <-ctx.Done():
			return CallbackResult{}, ctx.Err()
		case err := <-errCh:
			return CallbackResult{}, err
		case res := <-resultCh:
			return res, nil
		}
	}

	cleanup = func() error {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}

	return actualRedirect, wait, cleanup, nil
}
