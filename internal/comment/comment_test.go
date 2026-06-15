package comment

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const marker = "<!-- ephemeractl:cost-report -->"

// newTestPoster points a Poster at a stub GitHub API server.
func newTestPoster(t *testing.T, baseURL string) *Poster {
	t.Helper()
	p := NewPoster("tok", "acme", "checkout", 482)
	u := baseURL
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	if err := p.setBaseURL(u); err != nil {
		t.Fatalf("setBaseURL: %v", err)
	}
	return p
}

func TestUpsertCreatesWhenNoMarkerComment(t *testing.T) {
	var created bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/482/comments"):
			_, _ = w.Write([]byte(`[{"id":1,"body":"unrelated"}]`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/482/comments"):
			created = true
			_, _ = w.Write([]byte(`{"id":2,"html_url":"https://github.com/acme/checkout/pull/482#issuecomment-2"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	url, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nbody")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !created {
		t.Error("expected a POST (create) when no marker comment exists")
	}
	if !strings.Contains(url, "issuecomment-2") {
		t.Errorf("comment url = %q", url)
	}
}

func TestUpsertEditsExistingMarkerComment(t *testing.T) {
	var edited bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/482/comments"):
			body, _ := json.Marshal([]map[string]any{
				{"id": 7, "body": marker + "\nold"},
			})
			_, _ = w.Write(body)
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/comments/7"):
			edited = true
			_, _ = w.Write([]byte(`{"id":7,"html_url":"https://github.com/acme/checkout/pull/482#issuecomment-7"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	url, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nnew")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !edited {
		t.Error("expected a PATCH (edit) when a marker comment already exists")
	}
	if !strings.Contains(url, "issuecomment-7") {
		t.Errorf("comment url = %q", url)
	}
}
