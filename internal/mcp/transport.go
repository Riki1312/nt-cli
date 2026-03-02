package mcp

import "net/http"

const NotionMCPEndpoint = "https://mcp.notion.com/mcp"

// authRoundTripper injects an Authorization header into every request.
type authRoundTripper struct {
	token string
	inner http.RoundTripper
}

func (a *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+a.token)
	return a.inner.RoundTrip(req)
}

// NewAuthenticatedHTTPClient returns an http.Client that injects the given
// bearer token into every request.
func NewAuthenticatedHTTPClient(accessToken string) *http.Client {
	return &http.Client{
		Transport: &authRoundTripper{
			token: accessToken,
			inner: http.DefaultTransport,
		},
	}
}
