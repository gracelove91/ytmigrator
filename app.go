package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"ytmigrator/internal/state"
	"ytmigrator/internal/youtube"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// App struct holds application state including in-memory OAuth tokens.
type App struct {
	ctx         context.Context
	sourceToken *oauth2.Token // memory-only, never persisted
	targetToken *oauth2.Token // memory-only, never persisted
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// --- Credential storage helpers ---

func (a *App) configPath() string {
	cfgDir, _ := os.UserConfigDir()
	return filepath.Join(cfgDir, "ytmigrator", "credentials.json")
}

func (a *App) credentialsExist() bool {
	_, err := os.Stat(a.configPath())
	return err == nil
}

// GetStoredCredentialsStatus returns true if GCP credentials have been saved.
func (a *App) GetStoredCredentialsStatus() bool {
	return a.credentialsExist()
}

// SelectCredentialsFile opens a dialog, validates, and saves client_secret.json.
func (a *App) SelectCredentialsFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select GCP client_secret.json",
		Filters: []runtime.FileFilter{
			{DisplayName: "JSON Files (*.json)", Pattern: "*.json"},
		},
	})
	if err != nil {
		return "", fmt.Errorf("file dialog: %w", err)
	}
	if path == "" {
		return "", fmt.Errorf("no file selected")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	if _, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/youtube.readonly"); err != nil {
		return "", fmt.Errorf("invalid client_secret.json: %w", err)
	}

	configPath := a.configPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, b, 0600); err != nil {
		return "", fmt.Errorf("save credentials: %w", err)
	}
	return "saved", nil
}

// loadOAuthConfig reads stored credentials and builds an oauth2.Config.
func (a *App) loadOAuthConfig() (*oauth2.Config, error) {
	b, err := os.ReadFile(a.configPath())
	if err != nil {
		return nil, fmt.Errorf("credentials not found: %w", err)
	}
	// Use a broad scope here; actual scopes are selected during authenticate().
	cfg, err := google.ConfigFromJSON(b,
		"https://www.googleapis.com/auth/youtube.readonly",
		"https://www.googleapis.com/auth/youtube.force-ssl",
	)
	if err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}
	return cfg, nil
}

// AuthenticateSource performs OAuth with read-only scope.
// Returns a status string and stores the token in memory.
func (a *App) AuthenticateSource() (string, error) {
	tok, err := a.authenticate("https://www.googleapis.com/auth/youtube.readonly")
	if err != nil {
		return "", err
	}
	a.sourceToken = tok
	return "source authenticated", nil
}

// AuthenticateTarget performs OAuth with write scope.
// Returns a status string and stores the token in memory.
func (a *App) AuthenticateTarget() (string, error) {
	tok, err := a.authenticate("https://www.googleapis.com/auth/youtube.force-ssl")
	if err != nil {
		return "", err
	}
	a.targetToken = tok
	return "target authenticated", nil
}

func (a *App) authenticate(scope string) (*oauth2.Token, error) {
	cfg, err := a.loadOAuthConfig()
	if err != nil {
		return nil, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	cfg.RedirectURL = redirectURL

	authURL := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	runtime.BrowserOpenURL(a.ctx, authURL)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	server := &http.Server{}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no code in callback")
			http.Error(w, "Authentication failed", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "<h1>Success!</h1><p>You can close this window.</p>")
		codeCh <- code
		server.Close()
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case code := <-codeCh:
		tok, err := cfg.Exchange(a.ctx, code)
		if err != nil {
			return nil, fmt.Errorf("exchange token: %w", err)
		}
		return tok, nil
	case err := <-errCh:
		return nil, err
	}
}

// ExportData exports subscriptions, playlists, and liked videos from the source account.
// Requires AuthenticateSource() to have been called beforehand.
func (a *App) ExportData() (string, error) {
	if a.sourceToken == nil {
		return "", fmt.Errorf("source account not authenticated. call AuthenticateSource() first")
	}

	cfg, err := a.loadOAuthConfig()
	if err != nil {
		return "", err
	}
	httpClient := cfg.Client(a.ctx, a.sourceToken)

	client, err := youtube.NewClient(a.ctx, httpClient)
	if err != nil {
		return "", fmt.Errorf("create youtube client: %w", err)
	}

	bundle, err := client.ExportAll(a.ctx)
	if err != nil {
		return "", fmt.Errorf("export failed: %w", err)
	}

	exportPath := filepath.Join(os.TempDir(), "ytmigrator_export.json")
	if err := saveJSON(exportPath, bundle); err != nil {
		return "", fmt.Errorf("save export: %w", err)
	}
	return exportPath, nil
}

func saveJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// ImportData imports subscriptions, playlists, and liked videos into the target account.
// Requires AuthenticateTarget() to have been called beforehand.
func (a *App) ImportData(exportPath string) (string, error) {
	if a.targetToken == nil {
		return "", fmt.Errorf("target account not authenticated. call AuthenticateTarget() first")
	}

	// Load export bundle
	b, err := os.ReadFile(exportPath)
	if err != nil {
		return "", fmt.Errorf("read export file: %w", err)
	}
	var bundle youtube.ExportBundle
	if err := json.Unmarshal(b, &bundle); err != nil {
		return "", fmt.Errorf("parse export file: %w", err)
	}

	cfg, err := a.loadOAuthConfig()
	if err != nil {
		return "", err
	}
	httpClient := cfg.Client(a.ctx, a.targetToken)

	client, err := youtube.NewClient(a.ctx, httpClient)
	if err != nil {
		return "", fmt.Errorf("create youtube client: %w", err)
	}

	// Load or create progress
	progressPath := filepath.Join(os.TempDir(), "ytmigrator_import_progress.json")
	prog, err := state.LoadImportProgress(progressPath)
	if err != nil {
		return "", fmt.Errorf("load progress: %w", err)
	}

	// Run import
	if err := client.ImportAll(a.ctx, &bundle, prog); err != nil {
		_ = state.SaveImportProgress(progressPath, prog)
		return "", fmt.Errorf("import failed: %w", err)
	}

	if err := state.SaveImportProgress(progressPath, prog); err != nil {
		return "", fmt.Errorf("save progress: %w", err)
	}

	return fmt.Sprintf("import complete. %d subscriptions, %d playlists done, %d liked videos done",
		len(prog.SubscriptionsCompleted), len(prog.PlaylistsCompleted), len(prog.LikedVideosCompleted)), nil
}

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
