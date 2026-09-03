package openid

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestListenRedirectFixedPort(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	const redirectURL = "http://127.0.0.1:18765/callback"
	redirect, wait, cleanup, err := ListenRedirect(ctx, redirectURL)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if redirect != redirectURL {
		t.Fatalf("redirect=%q", redirect)
	}

	done := make(chan error, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		resp, err := http.Get(redirectURL + "?code=abc&state=xyz")
		if err != nil {
			done <- err
			return
		}
		resp.Body.Close()
		done <- nil
	}()

	res, err := wait()
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("GET: %v", err)
	}
	if res.Code != "abc" || res.State != "xyz" {
		t.Fatalf("%+v", res)
	}
}
