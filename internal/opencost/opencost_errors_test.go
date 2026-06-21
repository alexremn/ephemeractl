package opencost

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchErrorsOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := New(srv.URL).Fetch(context.Background(), Query{Window: "1h", LabelKey: "k", LabelValue: "1"})
	if err == nil {
		t.Fatal("expected error on HTTP 500, got nil")
	}
}

func TestFetchErrorsOnMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Fetch(context.Background(), Query{Window: "1h", LabelKey: "k", LabelValue: "1"})
	if err == nil {
		t.Fatal("expected a decode error on malformed JSON, got nil")
	}
}

func TestFetchErrorsOnBodyCode(t *testing.T) {
	// OpenCost reports application errors as HTTP 200 with a non-200 body code.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":400,"data":null}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL).Fetch(context.Background(), Query{Window: "1h", LabelKey: "k", LabelValue: "1"})
	if err == nil {
		t.Fatal("expected error on body-level code 400, got nil")
	}
}

func TestFetchHandlesEmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
	}))
	defer srv.Close()

	res, err := New(srv.URL).Fetch(context.Background(), Query{Window: "1h", LabelKey: "k", LabelValue: "1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 0 || len(res.Groups) != 0 {
		t.Errorf("empty data should yield zero result, got %+v", res)
	}
}
