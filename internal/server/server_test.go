package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"canasta-server/internal/room"
	"canasta-server/internal/server"
)

func TestCORSAllowsAnyOriginWhenUnconfigured(t *testing.T) {
	srv := server.New(room.NewManager(), "")
	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/rooms", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Origin", "http://localhost:5173")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("expected Access-Control-Allow-Origin: *, got %q", got)
	}
}

func TestCORSEchoesConfiguredOrigin(t *testing.T) {
	srv := server.New(room.NewManager(), "https://landanfagan.com")
	ts := httptest.NewServer(srv.Routes())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/rooms", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Origin", "https://landanfagan.com")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://landanfagan.com" {
		t.Errorf("expected Access-Control-Allow-Origin: https://landanfagan.com, got %q", got)
	}
}
