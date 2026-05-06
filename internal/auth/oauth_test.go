package auth

import (
	"net/url"
	"testing"
	"time"
)

func TestBuildWellKnownURL(t *testing.T) {
	t.Parallel()

	got, err := buildWellKnownURL("https://mcp.notion.com/mcp", "oauth-protected-resource")
	if err != nil {
		t.Fatalf("buildWellKnownURL returned error: %v", err)
	}

	want := "https://mcp.notion.com/.well-known/oauth-protected-resource/mcp"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildAuthURL(t *testing.T) {
	t.Parallel()

	got := buildAuthURL("https://auth.example/authorize", "client", "http://127.0.0.1/callback", "challenge", "state")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parsing URL: %v", err)
	}
	if got, want := u.Scheme+"://"+u.Host+u.Path, "https://auth.example/authorize"; got != want {
		t.Fatalf("endpoint = %q, want %q", got, want)
	}

	values := u.Query()
	want := map[string]string{
		"response_type":         "code",
		"client_id":             "client",
		"redirect_uri":          "http://127.0.0.1/callback",
		"state":                 "state",
		"code_challenge":        "challenge",
		"code_challenge_method": "S256",
	}
	for key, wantValue := range want {
		if gotValue := values.Get(key); gotValue != wantValue {
			t.Fatalf("%s = %q, want %q", key, gotValue, wantValue)
		}
	}
}

func TestTokenFromResponseUsesFallbackRefreshToken(t *testing.T) {
	t.Parallel()

	before := time.Now()
	tok := tokenFromResponse(&tokenResponse{
		AccessToken: "access",
		TokenType:   "Bearer",
		ExpiresIn:   60,
	}, "refresh")

	if tok.AccessToken != "access" {
		t.Fatalf("AccessToken = %q, want access", tok.AccessToken)
	}
	if tok.RefreshToken != "refresh" {
		t.Fatalf("RefreshToken = %q, want refresh", tok.RefreshToken)
	}
	if tok.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want Bearer", tok.TokenType)
	}
	if !tok.Expiry.After(before) {
		t.Fatalf("Expiry = %v, want after %v", tok.Expiry, before)
	}
}
