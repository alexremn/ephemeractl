package ghevent

import (
	"os"
	"path/filepath"
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
