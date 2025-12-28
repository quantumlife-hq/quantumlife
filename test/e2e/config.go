//go:build e2e

// Package e2e provides end-to-end tests using real API credentials.
package e2e

import (
	"os"
	"testing"
)

// E2EConfig holds credentials for E2E tests.
type E2EConfig struct {
	// GitHub
	GitHubToken     string
	GitHubTestOwner string
	GitHubTestRepo  string

	// Slack
	SlackToken       string
	SlackTestChannel string

	// Notion
	NotionToken        string
	NotionTestDatabase string

	// Gmail/Calendar (OAuth tokens as JSON)
	GmailCredentialsJSON    string
	GmailTokenJSON          string
	CalendarCredentialsJSON string
	CalendarTokenJSON       string

	// Plaid (Finance)
	PlaidClientID    string
	PlaidSecret      string
	PlaidEnv         string
	PlaidAccessToken string
}

// LoadE2EConfig loads configuration from environment variables.
func LoadE2EConfig(t *testing.T) *E2EConfig {
	t.Helper()

	return &E2EConfig{
		// GitHub
		GitHubToken:     os.Getenv("GITHUB_TOKEN"),
		GitHubTestOwner: os.Getenv("GITHUB_TEST_OWNER"),
		GitHubTestRepo:  os.Getenv("GITHUB_TEST_REPO"),

		// Slack
		SlackToken:       os.Getenv("SLACK_BOT_TOKEN"),
		SlackTestChannel: os.Getenv("SLACK_TEST_CHANNEL"),

		// Notion
		NotionToken:        os.Getenv("NOTION_API_KEY"),
		NotionTestDatabase: os.Getenv("NOTION_TEST_DATABASE"),

		// Gmail/Calendar
		GmailCredentialsJSON:    os.Getenv("GMAIL_CREDENTIALS_JSON"),
		GmailTokenJSON:          os.Getenv("GMAIL_TOKEN_JSON"),
		CalendarCredentialsJSON: os.Getenv("CALENDAR_CREDENTIALS_JSON"),
		CalendarTokenJSON:       os.Getenv("CALENDAR_TOKEN_JSON"),

		// Plaid
		PlaidClientID:    os.Getenv("PLAID_CLIENT_ID"),
		PlaidSecret:      os.Getenv("PLAID_SECRET"),
		PlaidEnv:         os.Getenv("PLAID_ENV"),
		PlaidAccessToken: os.Getenv("PLAID_ACCESS_TOKEN"),
	}
}

// RequireGitHub skips test if GitHub credentials are not available.
func (c *E2EConfig) RequireGitHub(t *testing.T) {
	t.Helper()
	if c.GitHubToken == "" {
		t.Skip("GITHUB_TOKEN not set, skipping GitHub E2E tests")
	}
}

// RequireGitHubRepo skips test if GitHub repo info is not available.
func (c *E2EConfig) RequireGitHubRepo(t *testing.T) {
	t.Helper()
	c.RequireGitHub(t)
	if c.GitHubTestOwner == "" || c.GitHubTestRepo == "" {
		t.Skip("GITHUB_TEST_OWNER and GITHUB_TEST_REPO required, skipping")
	}
}

// RequireSlack skips test if Slack credentials are not available.
func (c *E2EConfig) RequireSlack(t *testing.T) {
	t.Helper()
	if c.SlackToken == "" {
		t.Skip("SLACK_BOT_TOKEN not set, skipping Slack E2E tests")
	}
}

// RequireSlackChannel skips test if Slack channel is not available.
func (c *E2EConfig) RequireSlackChannel(t *testing.T) {
	t.Helper()
	c.RequireSlack(t)
	if c.SlackTestChannel == "" {
		t.Skip("SLACK_TEST_CHANNEL not set, skipping")
	}
}

// RequireNotion skips test if Notion credentials are not available.
func (c *E2EConfig) RequireNotion(t *testing.T) {
	t.Helper()
	if c.NotionToken == "" {
		t.Skip("NOTION_API_KEY not set, skipping Notion E2E tests")
	}
}

// RequireNotionDatabase skips test if Notion database is not available.
func (c *E2EConfig) RequireNotionDatabase(t *testing.T) {
	t.Helper()
	c.RequireNotion(t)
	if c.NotionTestDatabase == "" {
		t.Skip("NOTION_TEST_DATABASE not set, skipping")
	}
}

// RequireGmail skips test if Gmail credentials are not available.
func (c *E2EConfig) RequireGmail(t *testing.T) {
	t.Helper()
	if c.GmailCredentialsJSON == "" || c.GmailTokenJSON == "" {
		t.Skip("Gmail credentials not set, skipping Gmail E2E tests")
	}
}

// RequireCalendar skips test if Calendar credentials are not available.
func (c *E2EConfig) RequireCalendar(t *testing.T) {
	t.Helper()
	if c.CalendarCredentialsJSON == "" || c.CalendarTokenJSON == "" {
		t.Skip("Calendar credentials not set, skipping Calendar E2E tests")
	}
}

// RequirePlaid skips test if Plaid credentials are not available.
func (c *E2EConfig) RequirePlaid(t *testing.T) {
	t.Helper()
	if c.PlaidClientID == "" || c.PlaidSecret == "" {
		t.Skip("Plaid credentials not set, skipping Finance E2E tests")
	}
}

// RequirePlaidAccess skips test if Plaid access token is not available.
func (c *E2EConfig) RequirePlaidAccess(t *testing.T) {
	t.Helper()
	c.RequirePlaid(t)
	if c.PlaidAccessToken == "" {
		t.Skip("PLAID_ACCESS_TOKEN not set, skipping")
	}
}
