package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	mcpEndpoint  = "https://mcp.notion.com/mcp"
	loginTimeout = 5 * time.Minute
)

// ServerMetadata holds the discovered OAuth endpoints.
type ServerMetadata struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint,omitempty"`
}

// protectedResourceMetadata is the response from the protected resource well-known endpoint.
type protectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
}

// Login performs the full OAuth flow: discovery, registration, PKCE auth, token exchange.
// It prints the authorization URL for the user to open manually.
func Login(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, loginTimeout)
	defer cancel()

	// 1. Discover OAuth endpoints
	fmt.Fprintln(logWriter(), "Discovering OAuth endpoints...")
	metadata, err := discoverMetadata(ctx)
	if err != nil {
		return fmt.Errorf("discovering OAuth metadata: %w", err)
	}

	// 2. Start local callback server (need the port before registering the client)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// 3. Register client with the exact redirect URI
	clientID, err := registerClient(ctx, metadata.RegistrationEndpoint, redirectURI)
	if err != nil {
		return fmt.Errorf("registering OAuth client: %w", err)
	}
	if err := SaveClientRegistration(&ClientRegistration{ClientID: clientID}); err != nil {
		return fmt.Errorf("saving client registration: %w", err)
	}

	// 4. Generate PKCE parameters
	verifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("generating PKCE verifier: %w", err)
	}
	challenge := computeCodeChallenge(verifier)
	state, err := generateState()
	if err != nil {
		return fmt.Errorf("generating state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := startCallbackServer(listener, state, codeCh, errCh)
	defer server.Shutdown(context.Background())

	// 5. Build and display authorization URL
	authURL := buildAuthURL(metadata.AuthorizationEndpoint, clientID, redirectURI, challenge, state)
	fmt.Fprintf(logWriter(), "\nOpen this URL in your browser to authorize nt-cli:\n\n")
	fmt.Fprintln(logWriter(), authURL)
	fmt.Fprintf(logWriter(), "\nWaiting for authorization...\n")

	// 6. Wait for callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return fmt.Errorf("authorization callback: %w", err)
	case <-ctx.Done():
		return fmt.Errorf("authorization timed out")
	}

	// 7. Exchange code for tokens
	tok, err := exchangeCode(ctx, metadata, clientID, code, verifier, redirectURI)
	if err != nil {
		return fmt.Errorf("exchanging authorization code: %w", err)
	}

	// 8. Save tokens
	if err := SaveToken(tok); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	fmt.Fprintln(logWriter(), "Login successful.")
	return nil
}

func discoverMetadata(ctx context.Context) (*ServerMetadata, error) {
	// Step 1: Get protected resource metadata
	// RFC 9728: insert /.well-known/oauth-protected-resource between host and path
	prURL, err := buildWellKnownURL(mcpEndpoint, "oauth-protected-resource")
	if err != nil {
		return nil, fmt.Errorf("building discovery URL: %w", err)
	}
	prMeta, err := httpGetJSON[protectedResourceMetadata](ctx, prURL)
	if err != nil {
		return nil, fmt.Errorf("fetching protected resource metadata: %w", err)
	}
	if len(prMeta.AuthorizationServers) == 0 {
		return nil, fmt.Errorf("no authorization servers found")
	}

	// Step 2: Get authorization server metadata
	authServer := strings.TrimRight(prMeta.AuthorizationServers[0], "/")
	asURL := authServer + "/.well-known/oauth-authorization-server"
	metadata, err := httpGetJSON[ServerMetadata](ctx, asURL)
	if err != nil {
		return nil, fmt.Errorf("fetching authorization server metadata: %w", err)
	}

	return metadata, nil
}

func registerClient(ctx context.Context, endpoint, redirectURI string) (string, error) {
	body := map[string]any{
		"client_name":                 "nt-cli",
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	}
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshaling registration body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(data)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("registration request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("registration failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ClientID string `json:"client_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing registration response: %w", err)
	}
	if result.ClientID == "" {
		return "", fmt.Errorf("registration response missing client_id")
	}

	return result.ClientID, nil
}

func startCallbackServer(listener net.Listener, expectedState string, codeCh chan<- string, errCh chan<- error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("%s: %s", errMsg, desc)
			fmt.Fprintf(w, "Authorization failed: %s. You can close this tab.", desc)
			return
		}

		if r.URL.Query().Get("state") != expectedState {
			errCh <- fmt.Errorf("state mismatch in OAuth callback")
			fmt.Fprint(w, "Error: state mismatch. You can close this tab.")
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code in callback")
			fmt.Fprint(w, "Error: no authorization code received. You can close this tab.")
			return
		}
		codeCh <- code
		fmt.Fprint(w, "Authorization successful! You can close this tab.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	return server
}

func buildAuthURL(endpoint, clientID, redirectURI, challenge, state string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return endpoint + "?" + params.Encode()
}

func exchangeCode(ctx context.Context, metadata *ServerMetadata, clientID, code, verifier, redirectURI string) (*Token, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {verifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	tok := &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		TokenType:    tokenResp.TokenType,
	}
	if tokenResp.ExpiresIn > 0 {
		tok.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return tok, nil
}

// RefreshToken attempts to refresh the access token using the refresh token.
func RefreshToken(ctx context.Context, tok *Token) (*Token, error) {
	if tok.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available; run 'nt login'")
	}

	metadata, err := discoverMetadata(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering OAuth metadata for refresh: %w", err)
	}

	reg, err := LoadClientRegistration()
	if err != nil || reg == nil {
		return nil, fmt.Errorf("no client registration found; run 'nt login'")
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {tok.RefreshToken},
		"client_id":     {reg.ClientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, metadata.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (HTTP %d): %s; run 'nt login'", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	newTok := &Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
	}
	// Use new refresh token if provided (rotation), otherwise keep the old one
	if tokenResp.RefreshToken != "" {
		newTok.RefreshToken = tokenResp.RefreshToken
	} else {
		newTok.RefreshToken = tok.RefreshToken
	}
	if tokenResp.ExpiresIn > 0 {
		newTok.Expiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	if err := SaveToken(newTok); err != nil {
		return nil, fmt.Errorf("saving refreshed token: %w", err)
	}

	return newTok, nil
}

// EnsureValidToken loads the token and refreshes it if expired.
func EnsureValidToken(ctx context.Context) (*Token, error) {
	tok, err := LoadToken()
	if err != nil {
		return nil, err
	}
	if !tok.IsExpired() {
		return tok, nil
	}
	return RefreshToken(ctx, tok)
}

// PKCE helpers

func generateCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func computeCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// buildWellKnownURL constructs a well-known URL per RFC 9728/RFC 8414.
// For "https://mcp.notion.com/mcp" with suffix "oauth-protected-resource",
// it produces "https://mcp.notion.com/.well-known/oauth-protected-resource/mcp".
func buildWellKnownURL(endpoint, suffix string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	path := u.Path
	u.Path = "/.well-known/" + suffix + path
	return u.String(), nil
}

func httpGetJSON[T any](ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func logWriter() io.Writer {
	return os.Stderr
}
