package core

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIEmptyChoicesReturnsError(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[]}`)
	}))
	defer srv.Close()
	o := &OpenAI{APIKey: "test", Model: "gpt-x", BaseURL: srv.URL}
	msgs := []Message{{Role: RoleUser, Content: "hi"}}

	// when
	_, err := o.Complete(context.Background(), msgs)

	// then
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no choices") {
		t.Errorf("error: got %v, want to contain %q", err, "no choices")
	}
}
