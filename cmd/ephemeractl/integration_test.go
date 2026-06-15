package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunEndToEndPostsCostComment exercises the whole run() pipeline against
// stub OpenCost and GitHub servers: event+input loading, window/selector
// resolution, allocation fetch, markdown render, sticky comment upsert, and
// $GITHUB_OUTPUT writing.
func TestRunEndToEndPostsCostComment(t *testing.T) {
	opencost := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/allocation" {
			t.Errorf("OpenCost path = %q, want /allocation", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":200,"data":[{"checkout":{"name":"checkout","cpuCost":1.0,"ramCost":0.5}}]}`))
	}))
	defer opencost.Close()

	var posted bool
	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/482/comments"):
			_, _ = w.Write([]byte(`[]`)) // no existing comment → create path
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/482/comments"):
			posted = true
			_, _ = w.Write([]byte(`{"id":1,"html_url":"https://example.test/pr/482#c1"}`))
		default:
			t.Errorf("unexpected GitHub call %s %s", r.Method, r.URL.Path)
		}
	}))
	defer githubAPI.Close()

	dir := t.TempDir()
	eventPath := filepath.Join(dir, "event.json")
	if err := os.WriteFile(eventPath, []byte(`{
	  "pull_request": { "number": 482, "created_at": "2026-06-01T10:00:00Z" },
	  "repository": { "name": "checkout", "owner": { "login": "acme" } }
	}`), 0o600); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(dir, "github_output")

	t.Setenv("GITHUB_EVENT_PATH", eventPath)
	t.Setenv("GITHUB_API_URL", githubAPI.URL)
	t.Setenv("GITHUB_OUTPUT", outPath)
	t.Setenv("INPUT_OPENCOST-URL", opencost.URL)
	t.Setenv("INPUT_GITHUB-TOKEN", "tok")
	t.Setenv("INPUT_TEAM-LABEL", "team")

	if code := run(); code != 0 {
		t.Fatalf("run() = %d, want 0", code)
	}
	if !posted {
		t.Error("expected run() to POST a cost comment")
	}
	out, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	if !strings.Contains(got, "total-cost=1.50") { // 1.0 cpu + 0.5 ram
		t.Errorf("GITHUB_OUTPUT missing total-cost=1.50:\n%s", got)
	}
	if !strings.Contains(got, "comment-url=https://example.test/pr/482#c1") {
		t.Errorf("GITHUB_OUTPUT missing comment-url:\n%s", got)
	}
}
