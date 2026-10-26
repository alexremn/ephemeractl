package ghevent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsNonHTTPOpenCostURL(t *testing.T) {
	t.Setenv("INPUT_OPENCOST-URL", "file:///etc/passwd")
	t.Setenv("GITHUB_EVENT_PATH", "/unused")

	if _, err := Load(); err == nil {
		t.Fatal("expected error for a non-http(s) opencost-url, got nil")
	}
}

func TestLoadRejectsZeroPRNumber(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "event.json")
	if err := os.WriteFile(path, []byte(`{
	  "pull_request": { "number": 0, "created_at": "2026-06-01T10:00:00Z" },
	  "repository": { "name": "r", "owner": { "login": "o" } }
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")
	t.Setenv("GITHUB_EVENT_PATH", path)

	if _, err := Load(); err == nil {
		t.Fatal("expected error for pull_request.number=0, got nil")
	}
}

func TestLoadErrorsOnUnreadableOrMalformedEvent(t *testing.T) {
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.json")
	if err := os.WriteFile(malformed, []byte(`{"pull_request":`), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, path, wantErr string
	}{
		{"missing file", filepath.Join(dir, "missing.json"), "read event payload"},
		{"malformed JSON", malformed, "parse event payload"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")
			t.Setenv("GITHUB_EVENT_PATH", tc.path)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Load() error = %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}
