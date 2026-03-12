package main

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerRoutesWithoutPasswordAllowsIndex(t *testing.T) {
	server := NewServer(&Typer{}, "")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 without password, got %d", rec.Code)
	}
}

func TestServerRoutesWithPasswordRejectsMissingAuth(t *testing.T) {
	server := NewServer(&Typer{}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got == "" {
		t.Fatalf("expected WWW-Authenticate header to be set")
	}
}

func TestServerRoutesWithPasswordAllowsMatchingPassword(t *testing.T) {
	server := NewServer(&Typer{}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:secret")))
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with correct password, got %d", rec.Code)
	}
}

func TestServerRoutesWithPasswordRejectsWrongPassword(t *testing.T) {
	server := NewServer(&Typer{}, "secret")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("user:wrong")))
	rec := httptest.NewRecorder()

	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong password, got %d", rec.Code)
	}
}
