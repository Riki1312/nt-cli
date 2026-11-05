package mcp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPClientDoesNotForwardCredentialsOrReplayRedirectedWrites(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		destinationCalls.Add(1)
	}))
	defer destination.Close()
	var calls atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer synthetic-token" || string(body) != "comment" {
			t.Error("missing credential or changed request body")
		}
		http.Redirect(w, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	request, err := http.NewRequest(http.MethodPost, origin.URL, strings.NewReader("comment"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := NewAuthenticatedHTTPClient("synthetic-token").Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTemporaryRedirect || calls.Load() != 1 || destinationCalls.Load() != 0 {
		t.Fatalf("redirect followed: status=%d origin=%d destination=%d", response.StatusCode, calls.Load(), destinationCalls.Load())
	}
}

func TestHTTPClientPreservesUncertainWritesAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name string
		wait bool
	}{
		{"lost response", false},
		{"canceled", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if _, err := io.Copy(io.Discard, r.Body); err != nil {
					t.Error(err)
					return
				}
				if test.wait {
					<-release
					return
				}
				connection, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				connection.Close()
			}))
			defer server.Close()
			defer close(release)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, strings.NewReader("comment"))
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewAuthenticatedHTTPClient("synthetic-token").Do(request)
			if err == nil || calls.Load() != 1 {
				t.Fatalf("uncertain write repeated or hidden: calls=%d, error=%v", calls.Load(), err)
			}
			if test.wait && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("cancellation hidden: %v", err)
			}
		})
	}
}
