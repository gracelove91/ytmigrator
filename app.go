package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// App struct
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// configPath returns the path to the stored credentials file in the OS config directory.
func (a *App) configPath() string {
	cfgDir, _ := os.UserConfigDir()
	return filepath.Join(cfgDir, "ytmigrator", "credentials.json")
}

// credentialsExist checks if saved credentials are available.
func (a *App) credentialsExist() bool {
	_, err := os.Stat(a.configPath())
	return err == nil
}

// GetStoredCredentialsStatus returns whether credentials have been saved.
// Called by the frontend on app startup to show the correct UI state.
func (a *App) GetStoredCredentialsStatus() bool {
	return a.credentialsExist()
}

// SelectCredentialsFile opens a file dialog, validates the JSON,
// and saves the full client_secret.json content to the OS config directory.
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

	// Validate by parsing with google package
	_, err = google.ConfigFromJSON(b, "https://www.googleapis.com/auth/youtube.readonly")
	if err != nil {
		return "", fmt.Errorf("invalid client_secret.json: %w", err)
	}

	// Save to config directory
	configPath := a.configPath()
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(configPath, b, 0600); err != nil {
		return "", fmt.Errorf("save credentials: %w", err)
	}

	return "saved", nil
}

// AuthenticateSource authenticates with read-only scope using stored credentials.
func (a *App) AuthenticateSource() (string, error) {
	return a.authenticate("https://www.googleapis.com/auth/youtube.readonly")
}

// AuthenticateTarget authenticates with write scope using stored credentials.
func (a *App) AuthenticateTarget() (string, error) {
	return a.authenticate("https://www.googleapis.com/auth/youtube.force-ssl")
}

func (a *App) authenticate(scope string) (string, error) {
	b, err := os.ReadFile(a.configPath())
	if err != nil {
		return "", fmt.Errorf("stored credentials not found: %w", err)
	}

	config, err := google.ConfigFromJSON(b, scope)
	if err != nil {
		return "", fmt.Errorf("parse credentials: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline)

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
		tok, err := config.Exchange(a.ctx, code)
		if err != nil {
			return "", fmt.Errorf("exchange token: %w", err)
		}
		return tok.AccessToken, nil
	case err := <-errCh:
		return "", err
	}
}
