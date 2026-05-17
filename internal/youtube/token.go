package youtube

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// MemoryToken holds an access token in memory (never persisted to disk).
// When a refresh token exists this can auto-refresh.
type MemoryToken struct {
	token      *oauth2.Token
	oauth2Cfg  *oauth2.Config
	httpClient *http.Client
}

// NewMemoryToken exchanges an OAuth code for a token and stores it in memory.
// This should be called after the redirect callback receives the authorization code.
func NewMemoryToken(ctx context.Context, cfg *oauth2.Config, code string) (*MemoryToken, error) {
	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	mt := &MemoryToken{token: tok, oauth2Cfg: cfg}
	mt.httpClient = cfg.Client(ctx, tok)
	return mt, nil
}

// HTTPClient returns an http.Client that auto-refreshes the access token.
func (mt *MemoryToken) HTTPClient() *http.Client {
	return mt.httpClient
}

// NewHttpClientFromAccessToken creates a reusable http.Client from a pre-existing token.
func NewHttpClientFromAccessToken(ctx context.Context, cfg *oauth2.Config, tok *oauth2.Token) *http.Client {
	return cfg.Client(ctx, tok)
}
