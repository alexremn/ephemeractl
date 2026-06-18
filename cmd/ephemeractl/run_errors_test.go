package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func writeEvent(t *testing.T, number int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "event.json")
	body := `{
	  "pull_request": { "number": ` + strconv.Itoa(number) + `, "created_at": "2026-06-01T10:00:00Z" },
	  "repository": { "name": "r", "owner": { "login": "o" } }
	}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunFailsWhenEventMissing(t *testing.T) {
	t.Setenv("GITHUB_EVENT_PATH", "")
	t.Setenv("INPUT_OPENCOST-URL", "http://oc:9003")

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1 when GITHUB_EVENT_PATH is unset", code)
	}
}

func TestRunFailsWhenOpenCostErrors(t *testing.T) {
	oc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer oc.Close()

	t.Setenv("GITHUB_EVENT_PATH", writeEvent(t, 5))
	t.Setenv("INPUT_OPENCOST-URL", oc.URL)
	t.Setenv("INPUT_GITHUB-TOKEN", "tok")

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1 when OpenCost returns 500", code)
	}
}

func TestRunFailsWhenCommentErrors(t *testing.T) {
	oc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[{"r":{"name":"r","cpuCost":1.0}}]}`))
	}))
	defer oc.Close()
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusForbidden) // POST create → 403
	}))
	defer gh.Close()

	t.Setenv("GITHUB_EVENT_PATH", writeEvent(t, 9))
	t.Setenv("INPUT_OPENCOST-URL", oc.URL)
	t.Setenv("GITHUB_API_URL", gh.URL)
	t.Setenv("INPUT_GITHUB-TOKEN", "tok")

	if code := run(); code != 1 {
		t.Errorf("run() = %d, want 1 when the comment API errors", code)
	}
}
