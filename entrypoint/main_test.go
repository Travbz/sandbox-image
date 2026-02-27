package main

import (
	"os"
	"testing"
)

func TestLookupUser_Root(t *testing.T) {
	uid, gid, err := lookupUser("root")
	if err != nil {
		t.Fatalf("lookupUser(root) error = %v", err)
	}
	if uid != 0 {
		t.Errorf("uid = %d, want 0", uid)
	}
	if gid != 0 {
		t.Errorf("gid = %d, want 0", gid)
	}
}

func TestLookupUser_NotFound(t *testing.T) {
	_, _, err := lookupUser("nonexistent_user_12345")
	if err == nil {
		t.Fatal("lookupUser(nonexistent) expected error")
	}
}

func TestEnvStripping(t *testing.T) {
	// Simulate what the entrypoint does: set then unset control plane vars.
	os.Setenv("SESSION_TOKEN", "test-token")
	os.Setenv("CONTROL_PLANE_URL", "http://localhost:8090")
	os.Setenv("SESSION_ID", "sandbox-1")

	// Read into locals.
	token := os.Getenv("SESSION_TOKEN")
	cpURL := os.Getenv("CONTROL_PLANE_URL")
	sessID := os.Getenv("SESSION_ID")

	// Strip.
	os.Unsetenv("SESSION_TOKEN")
	os.Unsetenv("CONTROL_PLANE_URL")
	os.Unsetenv("SESSION_ID")

	// Verify locals have values.
	if token != "test-token" {
		t.Errorf("token = %q, want %q", token, "test-token")
	}
	if cpURL != "http://localhost:8090" {
		t.Errorf("cpURL = %q, want %q", cpURL, "http://localhost:8090")
	}
	if sessID != "sandbox-1" {
		t.Errorf("sessID = %q, want %q", sessID, "sandbox-1")
	}

	// Verify env is clean.
	if v := os.Getenv("SESSION_TOKEN"); v != "" {
		t.Errorf("SESSION_TOKEN should be empty after unset, got %q", v)
	}
	if v := os.Getenv("CONTROL_PLANE_URL"); v != "" {
		t.Errorf("CONTROL_PLANE_URL should be empty after unset, got %q", v)
	}
	if v := os.Getenv("SESSION_ID"); v != "" {
		t.Errorf("SESSION_ID should be empty after unset, got %q", v)
	}
}
