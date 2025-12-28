//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/client/microsoft"
)

// getMicrosoft365Token returns the OAuth token from env var, or empty string to use cached credentials.
func getMicrosoft365Token() string {
	return os.Getenv("MS365_OAUTH_TOKEN")
}

// skipIfNoMicrosoft365 checks if Microsoft 365 tests should run.
// Tests will use cached credentials from `npx @softeria/ms-365-mcp-server --login`
func skipIfNoMicrosoft365(t *testing.T) {
	t.Helper()
	if os.Getenv("SKIP_MS365_TESTS") == "true" {
		t.Skip("SKIP_MS365_TESTS=true, skipping Microsoft 365 E2E tests")
	}
}

func TestMicrosoft365_E2E_Connect(t *testing.T) {
	skipIfNoMicrosoft365(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: getMicrosoft365Token(), // Uses cached credentials if empty
		Preset:     "personal",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	t.Log("Successfully connected to Microsoft 365 MCP server")
}

func TestMicrosoft365_E2E_ListTools(t *testing.T) {
	skipIfNoMicrosoft365(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: getMicrosoft365Token(),
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Microsoft 365 MCP server has %d tools", len(tools))

	// Log first 10 tools
	for i, tool := range tools {
		if i >= 10 {
			t.Logf("  ... and %d more tools", len(tools)-10)
			break
		}
		t.Logf("  - %s", tool.Name)
	}

	// Verify we have key tools
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	expectedTools := []string{"list-mail-messages", "list-calendars", "list-drives"}
	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("Expected tool %q not found", expected)
		}
	}
}

func TestMicrosoft365_E2E_GetCurrentUser(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		// No preset - get-current-user is a general tool
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.GetCurrentUser(ctx)
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("GetCurrentUser returned error: %s", result.Content[0].Text)
	}

	t.Logf("Current user: %s", truncate(result.Content[0].Text, 300))
}

func TestMicrosoft365_E2E_ListMailMessages(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "mail",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListMailMessages(ctx, 5)
	if err != nil {
		t.Fatalf("ListMailMessages failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListMailMessages returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed mail messages")
}

func TestMicrosoft365_E2E_ListMailFolders(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "mail",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListMailFolders(ctx)
	if err != nil {
		t.Fatalf("ListMailFolders failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListMailFolders returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed mail folders")
}

func TestMicrosoft365_E2E_ListCalendars(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "calendar",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListCalendars(ctx)
	if err != nil {
		t.Fatalf("ListCalendars failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListCalendars returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed calendars")
}

func TestMicrosoft365_E2E_ListCalendarEvents(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "calendar",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListCalendarEvents(ctx, "", 5)
	if err != nil {
		t.Fatalf("ListCalendarEvents failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListCalendarEvents returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed calendar events")
}

func TestMicrosoft365_E2E_ListDrives(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "files",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListDrives(ctx)
	if err != nil {
		t.Fatalf("ListDrives failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListDrives returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed OneDrive drives")
}

func TestMicrosoft365_E2E_ListFolderFiles(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "files",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First get the drive ID
	driveResult, err := client.ListDrives(ctx)
	if err != nil {
		t.Fatalf("ListDrives failed: %v", err)
	}
	if driveResult.IsError {
		t.Fatalf("ListDrives returned error: %s", driveResult.Content[0].Text)
	}

	// Parse the response to get drive ID
	var drivesData map[string]interface{}
	if err := json.Unmarshal([]byte(driveResult.Content[0].Text), &drivesData); err != nil {
		t.Fatalf("Failed to parse drives response: %v", err)
	}

	// Extract first drive ID from response
	var driveID string
	if value, ok := drivesData["value"].([]interface{}); ok && len(value) > 0 {
		if drive, ok := value[0].(map[string]interface{}); ok {
			if id, ok := drive["id"].(string); ok {
				driveID = id
			}
		}
	}

	if driveID == "" {
		t.Skip("No drives found to test folder listing")
	}

	t.Logf("Using drive ID: %s", driveID)

	// Get the root item of the drive
	result, err := client.GetDriveRootItem(ctx, driveID)
	if err != nil {
		t.Fatalf("GetDriveRootItem failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("GetDriveRootItem returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully got OneDrive root item")
}

func TestMicrosoft365_E2E_ListTodoTaskLists(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "tasks",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListTodoTaskLists(ctx)
	if err != nil {
		t.Fatalf("ListTodoTaskLists failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListTodoTaskLists returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed To-Do task lists")
}

func TestMicrosoft365_E2E_ListContacts(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
		Preset:     "contacts",
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.ListContacts(ctx, 5)
	if err != nil {
		t.Fatalf("ListContacts failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("ListContacts returned error: %s", result.Content[0].Text)
	}

	t.Log("Successfully listed contacts")
}

func TestMicrosoft365_E2E_FullFlow(t *testing.T) {
	skipIfNoMicrosoft365(t)
	token := getMicrosoft365Token()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := microsoft.New(microsoft.Options{
		OAuthToken: token,
	})
	if err != nil {
		t.Fatalf("Failed to create Microsoft 365 client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	t.Log("Step 1: Connected to Microsoft 365 MCP server")

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}
	t.Logf("Step 2: Found %d tools", len(tools))

	// Get current user
	userResult, err := client.GetCurrentUser(ctx)
	if err != nil {
		t.Logf("Step 3: GetCurrentUser failed: %v", err)
	} else if userResult.IsError {
		t.Logf("Step 3: GetCurrentUser error: %s", truncate(userResult.Content[0].Text, 100))
	} else {
		t.Log("Step 3: Got current user profile")
	}

	// List calendars
	calResult, err := client.ListCalendars(ctx)
	if err != nil {
		t.Logf("Step 4: ListCalendars failed: %v", err)
	} else if calResult.IsError {
		t.Logf("Step 4: ListCalendars error: %s", truncate(calResult.Content[0].Text, 100))
	} else {
		t.Log("Step 4: Listed calendars")
	}

	// List drives
	driveResult, err := client.ListDrives(ctx)
	if err != nil {
		t.Logf("Step 5: ListDrives failed: %v", err)
	} else if driveResult.IsError {
		t.Logf("Step 5: ListDrives error: %s", truncate(driveResult.Content[0].Text, 100))
	} else {
		t.Log("Step 5: Listed OneDrive drives")
	}

	t.Log("Full Microsoft 365 flow completed!")
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
