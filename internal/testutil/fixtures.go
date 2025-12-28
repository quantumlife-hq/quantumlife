package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// RandomID generates a random ID for testing.
func RandomID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// EmailFixture creates a test email fixture.
type EmailFixture struct {
	ID        string
	From      string
	To        []string
	Subject   string
	Body      string
	Date      time.Time
	Labels    []string
	IsUnread  bool
	IsStarred bool
}

// DefaultEmailFixture returns a default email fixture.
func DefaultEmailFixture() EmailFixture {
	return EmailFixture{
		ID:        "email-" + RandomID(),
		From:      "sender@example.com",
		To:        []string{"recipient@example.com"},
		Subject:   "Test Email Subject",
		Body:      "This is the test email body content.",
		Date:      time.Now(),
		Labels:    []string{"INBOX"},
		IsUnread:  true,
		IsStarred: false,
	}
}

// CalendarEventFixture creates a test calendar event fixture.
type CalendarEventFixture struct {
	ID          string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	Attendees   []string
	AllDay      bool
}

// DefaultCalendarEventFixture returns a default calendar event fixture.
func DefaultCalendarEventFixture() CalendarEventFixture {
	start := time.Now().Add(time.Hour)
	return CalendarEventFixture{
		ID:          "event-" + RandomID(),
		Summary:     "Test Meeting",
		Description: "This is a test meeting.",
		Location:    "Conference Room A",
		Start:       start,
		End:         start.Add(time.Hour),
		Attendees:   []string{"attendee@example.com"},
		AllDay:      false,
	}
}

// GitHubIssueFixture creates a test GitHub issue fixture.
type GitHubIssueFixture struct {
	Number    int
	Title     string
	Body      string
	State     string
	Labels    []string
	User      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DefaultGitHubIssueFixture returns a default GitHub issue fixture.
func DefaultGitHubIssueFixture() GitHubIssueFixture {
	return GitHubIssueFixture{
		Number:    1,
		Title:     "Test Issue",
		Body:      "This is a test issue body.",
		State:     "open",
		Labels:    []string{"bug"},
		User:      "testuser",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
}

// GitHubRepoFixture creates a test GitHub repo fixture.
type GitHubRepoFixture struct {
	ID          int64
	Name        string
	FullName    string
	Description string
	Private     bool
	Stars       int
	Forks       int
	Language    string
}

// DefaultGitHubRepoFixture returns a default GitHub repo fixture.
func DefaultGitHubRepoFixture() GitHubRepoFixture {
	return GitHubRepoFixture{
		ID:          1,
		Name:        "test-repo",
		FullName:    "testuser/test-repo",
		Description: "A test repository",
		Private:     false,
		Stars:       42,
		Forks:       10,
		Language:    "Go",
	}
}

// SlackMessageFixture creates a test Slack message fixture.
type SlackMessageFixture struct {
	TS       string
	User     string
	Text     string
	ThreadTS string
	Channel  string
}

// DefaultSlackMessageFixture returns a default Slack message fixture.
func DefaultSlackMessageFixture() SlackMessageFixture {
	return SlackMessageFixture{
		TS:       "1234567890.123456",
		User:     "U1234567890",
		Text:     "Hello, world!",
		ThreadTS: "",
		Channel:  "C1234567890",
	}
}

// SlackChannelFixture creates a test Slack channel fixture.
type SlackChannelFixture struct {
	ID         string
	Name       string
	IsPrivate  bool
	NumMembers int
	Topic      string
	Purpose    string
}

// DefaultSlackChannelFixture returns a default Slack channel fixture.
func DefaultSlackChannelFixture() SlackChannelFixture {
	return SlackChannelFixture{
		ID:         "C1234567890",
		Name:       "general",
		IsPrivate:  false,
		NumMembers: 100,
		Topic:      "General discussion",
		Purpose:    "General channel for team communication",
	}
}

// NotionPageFixture creates a test Notion page fixture.
type NotionPageFixture struct {
	ID        string
	Title     string
	URL       string
	Archived  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// DefaultNotionPageFixture returns a default Notion page fixture.
func DefaultNotionPageFixture() NotionPageFixture {
	return NotionPageFixture{
		ID:        "page-" + RandomID(),
		Title:     "Test Page",
		URL:       "https://notion.so/test-page",
		Archived:  false,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
}

// TransactionFixture creates a test financial transaction fixture.
type TransactionFixture struct {
	ID           string
	AccountID    string
	Amount       float64
	Date         time.Time
	Name         string
	MerchantName string
	Category     []string
	Pending      bool
}

// DefaultTransactionFixture returns a default transaction fixture.
func DefaultTransactionFixture() TransactionFixture {
	return TransactionFixture{
		ID:           "txn-" + RandomID(),
		AccountID:    "acc-123",
		Amount:       -45.99,
		Date:         time.Now().Add(-2 * 24 * time.Hour),
		Name:         "AMAZON.COM",
		MerchantName: "Amazon",
		Category:     []string{"Shopping", "Online"},
		Pending:      false,
	}
}

// AccountFixture creates a test financial account fixture.
type AccountFixture struct {
	ID              string
	Name            string
	Type            string
	Subtype         string
	Balance         float64
	Currency        string
	InstitutionName string
}

// DefaultAccountFixture returns a default account fixture.
func DefaultAccountFixture() AccountFixture {
	return AccountFixture{
		ID:              "acc-" + RandomID(),
		Name:            "Checking Account",
		Type:            "depository",
		Subtype:         "checking",
		Balance:         1234.56,
		Currency:        "USD",
		InstitutionName: "Test Bank",
	}
}

// EmailFixtureBuilder builds email fixtures with a fluent interface.
type EmailFixtureBuilder struct {
	fixture EmailFixture
}

// NewEmailBuilder creates a new email fixture builder.
func NewEmailBuilder() *EmailFixtureBuilder {
	return &EmailFixtureBuilder{fixture: DefaultEmailFixture()}
}

// WithFrom sets the from field.
func (b *EmailFixtureBuilder) WithFrom(from string) *EmailFixtureBuilder {
	b.fixture.From = from
	return b
}

// WithTo sets the to field.
func (b *EmailFixtureBuilder) WithTo(to []string) *EmailFixtureBuilder {
	b.fixture.To = to
	return b
}

// WithSubject sets the subject field.
func (b *EmailFixtureBuilder) WithSubject(subject string) *EmailFixtureBuilder {
	b.fixture.Subject = subject
	return b
}

// WithBody sets the body field.
func (b *EmailFixtureBuilder) WithBody(body string) *EmailFixtureBuilder {
	b.fixture.Body = body
	return b
}

// WithLabels sets the labels field.
func (b *EmailFixtureBuilder) WithLabels(labels []string) *EmailFixtureBuilder {
	b.fixture.Labels = labels
	return b
}

// AsUnread marks the email as unread.
func (b *EmailFixtureBuilder) AsUnread() *EmailFixtureBuilder {
	b.fixture.IsUnread = true
	return b
}

// AsRead marks the email as read.
func (b *EmailFixtureBuilder) AsRead() *EmailFixtureBuilder {
	b.fixture.IsUnread = false
	return b
}

// Build returns the built fixture.
func (b *EmailFixtureBuilder) Build() EmailFixture {
	return b.fixture
}

// NewIssueBuilder creates a new GitHub issue fixture builder.
func NewIssueBuilder() *IssueFixtureBuilder {
	return &IssueFixtureBuilder{fixture: DefaultGitHubIssueFixture()}
}

// IssueFixtureBuilder builds GitHub issue fixtures.
type IssueFixtureBuilder struct {
	fixture GitHubIssueFixture
}

// WithTitle sets the title.
func (b *IssueFixtureBuilder) WithTitle(title string) *IssueFixtureBuilder {
	b.fixture.Title = title
	return b
}

// WithBody sets the body.
func (b *IssueFixtureBuilder) WithBody(body string) *IssueFixtureBuilder {
	b.fixture.Body = body
	return b
}

// WithState sets the state.
func (b *IssueFixtureBuilder) WithState(state string) *IssueFixtureBuilder {
	b.fixture.State = state
	return b
}

// WithLabels sets the labels.
func (b *IssueFixtureBuilder) WithLabels(labels []string) *IssueFixtureBuilder {
	b.fixture.Labels = labels
	return b
}

// Build returns the built fixture.
func (b *IssueFixtureBuilder) Build() GitHubIssueFixture {
	return b.fixture
}
