package testutil

import (
	"context"
	"time"
)

// MockGmailClient implements a mock Gmail client for testing.
type MockGmailClient struct {
	ListMessagesFunc   func(ctx context.Context, query string, maxResults int64) ([]GmailMessageSummary, error)
	GetMessageFunc     func(ctx context.Context, messageID string) (*GmailMessage, error)
	SendMessageFunc    func(ctx context.Context, to, subject, body string, cc []string) (*GmailSentMessage, error)
	ReplyFunc          func(ctx context.Context, threadID, body string) (*GmailSentMessage, error)
	ArchiveFunc        func(ctx context.Context, messageID string) error
	TrashFunc          func(ctx context.Context, messageID string) error
	StarFunc           func(ctx context.Context, messageID string, star bool) error
	MarkReadFunc       func(ctx context.Context, messageID string, read bool) error
	LabelFunc          func(ctx context.Context, messageID string, addLabels, removeLabels []string) error
	ListLabelsFunc     func(ctx context.Context) ([]GmailLabel, error)
	CreateDraftFunc    func(ctx context.Context, to, subject, body string) (*GmailDraft, error)
}

// GmailMessageSummary represents a Gmail message summary.
type GmailMessageSummary struct {
	ID       string
	ThreadID string
}

// GmailMessage represents a full Gmail message.
type GmailMessage struct {
	ID        string
	ThreadID  string
	From      string
	To        []string
	Subject   string
	Body      string
	Date      time.Time
	Labels    []string
	IsUnread  bool
	IsStarred bool
}

// GmailSentMessage represents a sent Gmail message.
type GmailSentMessage struct {
	ID       string
	ThreadID string
}

// GmailLabel represents a Gmail label.
type GmailLabel struct {
	ID   string
	Name string
	Type string
}

// GmailDraft represents a Gmail draft.
type GmailDraft struct {
	ID string
}

// ListMessages calls the mock function if set.
func (m *MockGmailClient) ListMessages(ctx context.Context, query string, maxResults int64) ([]GmailMessageSummary, error) {
	if m.ListMessagesFunc != nil {
		return m.ListMessagesFunc(ctx, query, maxResults)
	}
	return nil, nil
}

// GetMessage calls the mock function if set.
func (m *MockGmailClient) GetMessage(ctx context.Context, messageID string) (*GmailMessage, error) {
	if m.GetMessageFunc != nil {
		return m.GetMessageFunc(ctx, messageID)
	}
	return nil, nil
}

// MockCalendarClient implements a mock Calendar client for testing.
type MockCalendarClient struct {
	ListEventsFunc    func(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]CalendarEvent, error)
	GetEventFunc      func(ctx context.Context, calendarID, eventID string) (*CalendarEvent, error)
	CreateEventFunc   func(ctx context.Context, calendarID string, event *CalendarEvent) (*CalendarEvent, error)
	QuickAddFunc      func(ctx context.Context, calendarID, text string) (*CalendarEvent, error)
	UpdateEventFunc   func(ctx context.Context, calendarID string, event *CalendarEvent) (*CalendarEvent, error)
	DeleteEventFunc   func(ctx context.Context, calendarID, eventID string) error
	ListCalendarsFunc func(ctx context.Context) ([]Calendar, error)
}

// CalendarEvent represents a calendar event.
type CalendarEvent struct {
	ID          string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	Attendees   []string
	AllDay      bool
}

// Calendar represents a calendar.
type Calendar struct {
	ID       string
	Summary  string
	Primary  bool
	TimeZone string
}

// ListEvents calls the mock function if set.
func (m *MockCalendarClient) ListEvents(ctx context.Context, calendarID string, timeMin, timeMax time.Time, maxResults int64) ([]CalendarEvent, error) {
	if m.ListEventsFunc != nil {
		return m.ListEventsFunc(ctx, calendarID, timeMin, timeMax, maxResults)
	}
	return nil, nil
}

// MockSlackClient implements a mock Slack client for testing.
type MockSlackClient struct {
	ListChannelsFunc  func(ctx context.Context, excludeArchived bool, limit int) ([]SlackChannel, error)
	GetMessagesFunc   func(ctx context.Context, channelID string, limit int) ([]SlackMessage, error)
	SendMessageFunc   func(ctx context.Context, channelID, text string, threadTS string) (*SlackMessageResponse, error)
	AddReactionFunc   func(ctx context.Context, channelID, timestamp, emoji string) error
	SearchFunc        func(ctx context.Context, query string, count int) ([]SlackSearchMatch, error)
	GetUserFunc       func(ctx context.Context, userID string) (*SlackUser, error)
	ListUsersFunc     func(ctx context.Context, limit int) ([]SlackUser, error)
	GetPermalinkFunc  func(ctx context.Context, channelID, timestamp string) (string, error)
}

// SlackChannel represents a Slack channel.
type SlackChannel struct {
	ID         string
	Name       string
	IsPrivate  bool
	NumMembers int
	Topic      string
	Purpose    string
}

// SlackMessage represents a Slack message.
type SlackMessage struct {
	TS         string
	User       string
	Text       string
	ThreadTS   string
	ReplyCount int
	Reactions  []SlackReaction
}

// SlackReaction represents a Slack reaction.
type SlackReaction struct {
	Name  string
	Count int
	Users []string
}

// SlackMessageResponse represents a sent Slack message response.
type SlackMessageResponse struct {
	Channel  string
	TS       string
	ThreadTS string
}

// SlackSearchMatch represents a Slack search result.
type SlackSearchMatch struct {
	Text      string
	User      string
	Channel   string
	TS        string
	Permalink string
}

// SlackUser represents a Slack user.
type SlackUser struct {
	ID          string
	Name        string
	RealName    string
	DisplayName string
	Email       string
	IsAdmin     bool
	IsBot       bool
	StatusText  string
	StatusEmoji string
}

// ListChannels calls the mock function if set.
func (m *MockSlackClient) ListChannels(ctx context.Context, excludeArchived bool, limit int) ([]SlackChannel, error) {
	if m.ListChannelsFunc != nil {
		return m.ListChannelsFunc(ctx, excludeArchived, limit)
	}
	return nil, nil
}

// MockNotionClient implements a mock Notion client for testing.
type MockNotionClient struct {
	SearchFunc        func(ctx context.Context, query string, filter string) ([]NotionSearchResult, error)
	GetPageFunc       func(ctx context.Context, pageID string) (*NotionPage, error)
	GetContentFunc    func(ctx context.Context, blockID string) ([]NotionBlock, error)
	CreatePageFunc    func(ctx context.Context, parentID string, title string, content string) (*NotionPage, error)
	UpdatePageFunc    func(ctx context.Context, pageID string, properties map[string]interface{}) (*NotionPage, error)
	QueryDatabaseFunc func(ctx context.Context, databaseID string, filter map[string]interface{}) ([]NotionPage, error)
	ListDatabasesFunc func(ctx context.Context) ([]NotionDatabase, error)
	GetDatabaseFunc   func(ctx context.Context, databaseID string) (*NotionDatabase, error)
	AddCommentFunc    func(ctx context.Context, pageID, text string) (*NotionComment, error)
	GetCommentsFunc   func(ctx context.Context, blockID string) ([]NotionComment, error)
}

// NotionSearchResult represents a Notion search result.
type NotionSearchResult struct {
	ID     string
	Type   string
	Title  string
	URL    string
	Parent string
}

// NotionPage represents a Notion page.
type NotionPage struct {
	ID         string
	Title      string
	URL        string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Archived   bool
	Properties map[string]interface{}
}

// NotionBlock represents a Notion block.
type NotionBlock struct {
	ID      string
	Type    string
	Content string
}

// NotionDatabase represents a Notion database.
type NotionDatabase struct {
	ID          string
	Title       string
	Description string
	URL         string
	Properties  map[string]interface{}
}

// NotionComment represents a Notion comment.
type NotionComment struct {
	ID        string
	Text      string
	CreatedAt time.Time
	Author    string
}

// Search calls the mock function if set.
func (m *MockNotionClient) Search(ctx context.Context, query string, filter string) ([]NotionSearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, filter)
	}
	return nil, nil
}

// MockGitHubClient implements a mock GitHub client for testing.
type MockGitHubClient struct {
	ListReposFunc      func(ctx context.Context, repoType string, sort string, limit int) ([]GitHubRepo, error)
	GetRepoFunc        func(ctx context.Context, owner, repo string) (*GitHubRepo, error)
	ListIssuesFunc     func(ctx context.Context, owner, repo, state string, labels []string, limit int) ([]GitHubIssue, error)
	GetIssueFunc       func(ctx context.Context, owner, repo string, number int) (*GitHubIssue, error)
	CreateIssueFunc    func(ctx context.Context, owner, repo, title, body string, labels []string) (*GitHubIssue, error)
	ListPRsFunc        func(ctx context.Context, owner, repo, state string, limit int) ([]GitHubPR, error)
	GetPRFunc          func(ctx context.Context, owner, repo string, number int) (*GitHubPR, error)
	NotificationsFunc  func(ctx context.Context, all bool, limit int) ([]GitHubNotification, error)
	GetUserFunc        func(ctx context.Context, username string) (*GitHubUser, error)
	SearchReposFunc    func(ctx context.Context, query, sort string, limit int) ([]GitHubRepo, int, error)
	SearchIssuesFunc   func(ctx context.Context, query, sort string, limit int) ([]GitHubIssue, int, error)
	GetContentsFunc    func(ctx context.Context, owner, repo, path string) ([]GitHubContent, error)
	AddCommentFunc     func(ctx context.Context, owner, repo string, number int, body string) (*GitHubComment, error)
}

// GitHubRepo represents a GitHub repository.
type GitHubRepo struct {
	ID          int64
	Name        string
	FullName    string
	Description string
	Private     bool
	HTMLURL     string
	CloneURL    string
	Stars       int
	Forks       int
	OpenIssues  int
	Language    string
	UpdatedAt   time.Time
}

// GitHubIssue represents a GitHub issue.
type GitHubIssue struct {
	Number    int
	Title     string
	Body      string
	State     string
	HTMLURL   string
	User      string
	Labels    []string
	Comments  int
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GitHubPR represents a GitHub pull request.
type GitHubPR struct {
	Number    int
	Title     string
	Body      string
	State     string
	HTMLURL   string
	User      string
	Head      string
	Base      string
	Merged    bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GitHubNotification represents a GitHub notification.
type GitHubNotification struct {
	ID        string
	Type      string
	Reason    string
	Unread    bool
	Subject   string
	Repo      string
	URL       string
	UpdatedAt time.Time
}

// GitHubUser represents a GitHub user.
type GitHubUser struct {
	Login     string
	Name      string
	Email     string
	Bio       string
	Company   string
	Location  string
	HTMLURL   string
	Repos     int
	Followers int
	Following int
}

// GitHubContent represents a GitHub repository content item.
type GitHubContent struct {
	Name    string
	Path    string
	Type    string
	Size    int64
	HTMLURL string
}

// GitHubComment represents a GitHub comment.
type GitHubComment struct {
	ID        int64
	Body      string
	User      string
	CreatedAt time.Time
}

// ListRepos calls the mock function if set.
func (m *MockGitHubClient) ListRepos(ctx context.Context, repoType string, sort string, limit int) ([]GitHubRepo, error) {
	if m.ListReposFunc != nil {
		return m.ListReposFunc(ctx, repoType, sort, limit)
	}
	return nil, nil
}

// MockFinanceSpace implements a mock Finance space for testing.
type MockFinanceSpace struct {
	IsConnectedFunc      func() bool
	ListAccountsFunc     func(ctx context.Context) ([]FinanceAccount, error)
	GetBalanceFunc       func(ctx context.Context) (*FinanceBalance, error)
	ListTransactionsFunc func(ctx context.Context, opts TransactionOptions) ([]FinanceTransaction, error)
	SpendingSummaryFunc  func(ctx context.Context, period string) (*SpendingSummary, error)
	RecurringFunc        func(ctx context.Context) ([]RecurringTransaction, error)
	InsightsFunc         func(ctx context.Context) ([]FinanceInsight, error)
	ConnectionsFunc      func(ctx context.Context) ([]FinanceConnection, error)
	SetBudgetFunc        func(ctx context.Context, category string, amount float64, period string) error
	GetBudgetsFunc       func(ctx context.Context) ([]Budget, error)
	CreateLinkTokenFunc  func(ctx context.Context) (string, error)
	SearchFunc           func(ctx context.Context, query string, limit int) ([]FinanceTransaction, error)
}

// FinanceAccount represents a financial account.
type FinanceAccount struct {
	ID             string
	Name           string
	Type           string
	Subtype        string
	Balance        float64
	Currency       string
	InstitutionID  string
	InstitutionName string
}

// FinanceBalance represents overall balance.
type FinanceBalance struct {
	NetWorth   float64
	TotalAssets float64
	TotalLiabilities float64
	Accounts   []FinanceAccount
}

// FinanceTransaction represents a financial transaction.
type FinanceTransaction struct {
	ID           string
	AccountID    string
	Amount       float64
	Date         time.Time
	Name         string
	MerchantName string
	Category     []string
	Pending      bool
}

// TransactionOptions represents options for listing transactions.
type TransactionOptions struct {
	StartDate time.Time
	EndDate   time.Time
	AccountID string
	Category  string
	MinAmount float64
	MaxAmount float64
	Limit     int
}

// SpendingSummary represents a spending summary.
type SpendingSummary struct {
	Period     string
	Total      float64
	Categories map[string]float64
}

// RecurringTransaction represents a recurring transaction.
type RecurringTransaction struct {
	ID        string
	Name      string
	Amount    float64
	Frequency string
	Category  string
	Active    bool
}

// FinanceInsight represents a financial insight.
type FinanceInsight struct {
	Type        string
	Title       string
	Description string
	Severity    string
}

// FinanceConnection represents a bank connection.
type FinanceConnection struct {
	ID             string
	InstitutionID  string
	InstitutionName string
	Status         string
	LastSync       time.Time
}

// Budget represents a budget.
type Budget struct {
	Category string
	Amount   float64
	Period   string
	Spent    float64
}

// IsConnected calls the mock function if set.
func (m *MockFinanceSpace) IsConnected() bool {
	if m.IsConnectedFunc != nil {
		return m.IsConnectedFunc()
	}
	return false
}
