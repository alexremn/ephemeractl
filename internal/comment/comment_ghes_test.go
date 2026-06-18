package comment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewPosterHonorsGitHubAPIURL(t *testing.T) {
	t.Setenv("GITHUB_API_URL", "https://ghe.example.com/api/v3")

	p := NewPoster("tok", "o", "r", 1)

	if got := p.gh.BaseURL.String(); got != "https://ghe.example.com/api/v3/" {
		t.Errorf("BaseURL = %q, want the GHES URL with a trailing slash", got)
	}
}

func TestUpsertSurfacesEditFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/482/comments"):
			body, _ := json.Marshal([]map[string]any{{"id": 7, "body": marker + "\nold"}})
			_, _ = w.Write(body)
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/comments/7"):
			w.WriteHeader(http.StatusUnprocessableEntity)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	_, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nnew")
	if err == nil || !strings.Contains(err.Error(), "edit sticky comment") {
		t.Fatalf("expected an 'edit sticky comment' error, got %v", err)
	}
}
