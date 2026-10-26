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

func TestUpsertErrorsWhenListCommentsFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nbody")
	if err == nil {
		t.Fatal("expected error when ListComments returns 500, got nil")
	}
	if !strings.Contains(err.Error(), "list PR comments") {
		t.Errorf("error message = %q, want it to contain 'list PR comments'", err.Error())
	}
}

func TestUpsertErrorsWhenCreateCommentFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/482/comments"):
			_, _ = w.Write([]byte(`[]`)) // no existing comments
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/482/comments"):
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	_, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nbody")
	if err == nil {
		t.Fatal("expected error when CreateComment returns 403, got nil")
	}
	if !strings.Contains(err.Error(), "create sticky comment") {
		t.Errorf("error message = %q, want it to contain 'create sticky comment'", err.Error())
	}
}

func TestSetBaseURLRejectsInvalidURL(t *testing.T) {
	p := NewPoster("tok", "acme", "repo", 1)
	// A URL with a control character is invalid and url.Parse returns an error.
	err := p.setBaseURL("http://host\x00/")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

// A busy PR can have more than one page of comments; the marker comment must
// still be found and edited, never duplicated.
func TestUpsertFindsMarkerOnLaterPage(t *testing.T) {
	var edited, created bool
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("page") == "":
			w.Header().Set("Link", `<`+srv.URL+r.URL.Path+`?page=2&per_page=100>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":1,"body":"unrelated"}]`))
		case r.Method == http.MethodGet && r.URL.Query().Get("page") == "2":
			body, _ := json.Marshal([]map[string]any{{"id": 9, "body": marker + "\nold"}})
			_, _ = w.Write(body)
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/issues/comments/9"):
			edited = true
			_, _ = w.Write([]byte(`{"id":9,"html_url":"https://github.com/acme/checkout/pull/482#issuecomment-9"}`))
		case r.Method == http.MethodPost:
			created = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":10}`))
		default:
			t.Errorf("unexpected %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
	}))
	defer srv.Close()

	url, err := newTestPoster(t, srv.URL).Upsert(context.Background(), marker, marker+"\nnew")
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if !edited || created {
		t.Errorf("edited=%v created=%v, want the page-2 marker comment edited and nothing created", edited, created)
	}
	if !strings.Contains(url, "issuecomment-9") {
		t.Errorf("comment url = %q", url)
	}
}
