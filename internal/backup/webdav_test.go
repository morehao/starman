package backup

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebDAVTestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PROPFIND" {
			t.Fatalf("expected PROPFIND, got %s", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Fatal("expected auth header")
		}
		w.WriteHeader(207)
	}))
	defer server.Close()
	c := NewWebDAVClient(server.URL, "user", "pass")
	if err := c.Test(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWebDAVPushPull(t *testing.T) {
	var pushedData []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body)
			pushedData = buf.Bytes()
			w.WriteHeader(201)
		case "GET":
			w.Write(pushedData)
		}
	}))
	defer server.Close()
	c := NewWebDAVClient(server.URL, "user", "pass")
	data := []byte(`{"test":true}`)
	if err := c.Push(context.Background(), "/backup.json", data); err != nil {
		t.Fatal(err)
	}
	got, err := c.Pull(context.Background(), "/backup.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %s, got %s", data, got)
	}
}
