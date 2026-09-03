package openid

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api2/json/access/openid/auth-url" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Form.Get("realm") != "my-realm" || r.Form.Get("redirect-url") != "http://127.0.0.1:8765/callback" {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":"https://idp.example.com/auth"}`)
	}))
	t.Cleanup(srv.Close)

	client := &Client{Endpoint: srv.URL}
	got, err := client.AuthURL(context.Background(), "my-realm", "http://127.0.0.1:8765/callback")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://idp.example.com/auth" {
		t.Fatalf("got %q", got)
	}
}

func TestLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api2/json/access/openid/login" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if r.Form.Get("code") != "abc" || r.Form.Get("state") != "xyz" {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"data":{"username":"user@idp","ticket":"ticket-1","CSRFPreventionToken":"csrf-1"}}`)
	}))
	t.Cleanup(srv.Close)

	client := &Client{Endpoint: srv.URL}
	sess, err := client.Login(context.Background(), "abc", "xyz", "http://127.0.0.1:8765/callback")
	if err != nil {
		t.Fatal(err)
	}
	if sess.Username != "user@idp" || sess.Ticket != "ticket-1" || sess.CSRFPreventionToken != "csrf-1" {
		t.Fatalf("%+v", sess)
	}
}

func TestAuthURLContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &Client{Endpoint: srv.URL}
	if _, err := client.AuthURL(ctx, "realm", "http://127.0.0.1/callback"); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyTicketOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/version" {
			http.NotFound(w, r)
			return
		}
		cookie, err := r.Cookie("PVEAuthCookie")
		if err != nil || cookie.Value != "good-ticket" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"version":"9.0"}}`))
	}))
	t.Cleanup(srv.Close)

	client := &Client{Endpoint: srv.URL}
	if err := client.VerifyTicket(context.Background(), "good-ticket"); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyTicketRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	client := &Client{Endpoint: srv.URL}
	if err := client.VerifyTicket(context.Background(), "bad-ticket"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAuthURLRejectsEmptyEndpoint(t *testing.T) {
	client := &Client{}
	if _, err := client.AuthURL(context.Background(), "realm", "http://127.0.0.1/callback"); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoginHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)

	client := &Client{Endpoint: srv.URL}
	_, err := client.Login(context.Background(), "c", "s", mustRedirectURL(t))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("err=%v", err)
	}
}

func mustRedirectURL(t *testing.T) string {
	t.Helper()
	u, err := url.Parse("http://127.0.0.1:8765/callback")
	if err != nil {
		t.Fatal(err)
	}
	return u.String()
}
