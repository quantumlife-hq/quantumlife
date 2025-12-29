//go:build ignore

// This script helps obtain Google OAuth tokens for E2E testing.
// Run with: go run scripts/get-google-token.go <credentials.json> [scope]
//
// Scopes:
//   gmail    - Gmail read/write access
//   calendar - Calendar read/write access
//   both     - Both Gmail and Calendar access (default)
//
// Supports both Desktop and Web Application OAuth credentials.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/get-google-token.go <credentials.json> [scope]")
		fmt.Println("Scopes: gmail, calendar, both (default: both)")
		os.Exit(1)
	}

	credFile := os.Args[1]
	scope := "both"
	if len(os.Args) > 2 {
		scope = os.Args[2]
	}

	// Read credentials
	credBytes, err := os.ReadFile(credFile)
	if err != nil {
		fmt.Printf("Error reading credentials: %v\n", err)
		os.Exit(1)
	}

	// Determine scopes
	var scopes []string
	switch scope {
	case "gmail":
		scopes = []string{
			gmail.GmailReadonlyScope,
			gmail.GmailModifyScope,
			gmail.GmailLabelsScope,
		}
	case "calendar":
		scopes = []string{
			calendar.CalendarReadonlyScope,
			calendar.CalendarEventsScope,
			calendar.CalendarScope,
		}
	case "both":
		scopes = []string{
			gmail.GmailReadonlyScope,
			gmail.GmailModifyScope,
			gmail.GmailLabelsScope,
			calendar.CalendarReadonlyScope,
			calendar.CalendarEventsScope,
			calendar.CalendarScope,
		}
	default:
		fmt.Printf("Unknown scope: %s\n", scope)
		os.Exit(1)
	}

	// Parse credentials
	config, err := google.ConfigFromJSON(credBytes, scopes...)
	if err != nil {
		fmt.Printf("Error parsing credentials: %v\n", err)
		os.Exit(1)
	}

	// For Desktop Application credentials, use loopback redirect
	// Find an available port dynamically
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("Error finding available port: %v\n", err)
		os.Exit(1)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Set redirect URI to loopback (Desktop OAuth standard)
	config.RedirectURL = fmt.Sprintf("http://127.0.0.1:%d", port)

	// Start local server for callback
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg != "" {
				errChan <- fmt.Errorf("OAuth error: %s", errMsg)
				http.Error(w, "Authorization failed: "+errMsg, http.StatusBadRequest)
				return
			}
			// Might be favicon or other request, ignore
			return
		}
		codeChan <- code
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html><html><body style="font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100vh;background:#4CAF50;color:white;"><div style="text-align:center"><h1>Success!</h1><p>You can close this window and return to the terminal.</p></div></body></html>`)
	})

	server := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", port), Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()
	defer server.Shutdown(context.Background())

	// Generate auth URL
	authURL := config.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("\n=== Google OAuth Setup ===")
	fmt.Printf("\nUsing redirect URI: %s\n", config.RedirectURL)
	fmt.Println("\nOpening browser for authentication...")

	// Try to open browser automatically
	if err := openBrowser(authURL); err != nil {
		fmt.Println("\nCould not open browser automatically.")
		fmt.Println("Please open this URL manually:\n")
		fmt.Println(authURL)
	}

	fmt.Println("\nWaiting for authorization...")

	// Wait for callback
	var code string
	select {
	case code = <-codeChan:
		fmt.Println("\nAuthorization received!")
	case err := <-errChan:
		fmt.Printf("\nError: %v\n", err)
		os.Exit(1)
	case <-time.After(5 * time.Minute):
		fmt.Println("\nTimeout waiting for authorization")
		os.Exit(1)
	}

	// Exchange code for token
	token, err := config.Exchange(context.Background(), code)
	if err != nil {
		fmt.Printf("Error exchanging code: %v\n", err)
		os.Exit(1)
	}

	// Output token as JSON
	tokenJSON, _ := json.MarshalIndent(token, "", "  ")
	tokenJSONCompact, _ := json.Marshal(token)
	credJSON := strings.ReplaceAll(string(credBytes), "\n", "")
	credJSON = strings.ReplaceAll(credJSON, "  ", "")

	fmt.Println("\n=== Success! ===\n")
	fmt.Println("Token JSON:")
	fmt.Println(string(tokenJSON))

	fmt.Println("\n=== Environment Variables for E2E Tests ===\n")
	fmt.Printf("export GMAIL_CREDENTIALS_JSON='%s'\n", credJSON)
	fmt.Printf("export GMAIL_TOKEN_JSON='%s'\n", string(tokenJSONCompact))
	fmt.Printf("export CALENDAR_CREDENTIALS_JSON='%s'\n", credJSON)
	fmt.Printf("export CALENDAR_TOKEN_JSON='%s'\n", string(tokenJSONCompact))

	fmt.Println("\n=== Or add to .env.e2e ===\n")
	fmt.Printf("GMAIL_CREDENTIALS_JSON=%s\n", credJSON)
	fmt.Printf("GMAIL_TOKEN_JSON=%s\n", string(tokenJSONCompact))
	fmt.Printf("CALENDAR_CREDENTIALS_JSON=%s\n", credJSON)
	fmt.Printf("CALENDAR_TOKEN_JSON=%s\n", string(tokenJSONCompact))
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try xdg-open first, then sensible-browser, then common browsers
		if _, err := exec.LookPath("xdg-open"); err == nil {
			cmd = exec.Command("xdg-open", url)
		} else if _, err := exec.LookPath("sensible-browser"); err == nil {
			cmd = exec.Command("sensible-browser", url)
		} else if _, err := exec.LookPath("firefox"); err == nil {
			cmd = exec.Command("firefox", url)
		} else if _, err := exec.LookPath("google-chrome"); err == nil {
			cmd = exec.Command("google-chrome", url)
		} else {
			return fmt.Errorf("no browser found")
		}
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}
