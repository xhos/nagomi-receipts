package vision

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAsk(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization = %q", got)
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		parts := req.Messages[0].Content
		if req.Model != "m" || parts[0].Text != "read this" || parts[1].ImageURL.URL != "data:image/png;base64,aW1n" {
			t.Errorf("unexpected request: %+v", req)
		}

		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer api.Close()

	reply, err := New(api.URL+"/v1/", "secret", "m").Ask(context.Background(), "read this", []byte("img"), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	if reply != "hello" {
		t.Errorf("reply = %q", reply)
	}
}

func TestErrorsIncludeAPIMessage(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"invalid api key"}}`, http.StatusUnauthorized)
	}))
	defer api.Close()

	err := New(api.URL, "", "m").Ping(context.Background())
	if err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Errorf("err = %v", err)
	}
}
