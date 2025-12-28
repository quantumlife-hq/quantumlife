// Package microsoft provides a client for the ms-365-mcp-server.
package microsoft

import (
	"context"
	"fmt"

	"github.com/quantumlife/quantumlife/internal/mcp/client"
)

// Client wraps the ms-365-mcp-server for Microsoft 365 integration.
type Client struct {
	*client.Client
	orgMode bool
}

// Options configures the Microsoft 365 client.
type Options struct {
	// OAuthToken is a pre-existing access token (BYOT mode)
	OAuthToken string
	// OrgMode enables organization-level tools (Teams, SharePoint, etc.)
	OrgMode bool
	// ReadOnly starts server in read-only mode
	ReadOnly bool
	// Preset limits tools to specific categories (mail, calendar, files, etc.)
	Preset string
}

// New creates a new Microsoft 365 MCP client using @softeria/ms-365-mcp-server.
func New(opts Options) (*Client, error) {
	args := []string{"-y", "@softeria/ms-365-mcp-server"}

	if opts.OrgMode {
		args = append(args, "--org-mode")
	}
	if opts.ReadOnly {
		args = append(args, "--read-only")
	}
	if opts.Preset != "" {
		args = append(args, "--preset", opts.Preset)
	}

	env := []string{}
	if opts.OAuthToken != "" {
		env = append(env, "MS365_MCP_OAUTH_TOKEN="+opts.OAuthToken)
	}

	c, err := client.New("npx", args, env)
	if err != nil {
		return nil, fmt.Errorf("failed to start Microsoft 365 MCP server: %w", err)
	}

	return &Client{
		Client:  c,
		orgMode: opts.OrgMode,
	}, nil
}

// Connect initializes the connection to the Microsoft 365 MCP server.
func (c *Client) Connect(ctx context.Context) error {
	return c.Initialize(ctx)
}

// ===== Email (Outlook) Tools =====

// ListMailMessages lists email messages from inbox.
func (c *Client) ListMailMessages(ctx context.Context, limit int) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if limit > 0 {
		args["top"] = limit
	}
	return c.CallTool(ctx, "list-mail-messages", args)
}

// ListMailFolders lists mail folders.
func (c *Client) ListMailFolders(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-mail-folders", map[string]interface{}{})
}

// GetMailMessage retrieves a specific email message.
func (c *Client) GetMailMessage(ctx context.Context, messageID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "get-mail-message", map[string]interface{}{
		"messageId": messageID,
	})
}

// SendMail sends an email.
func (c *Client) SendMail(ctx context.Context, to, subject, body string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "send-mail", map[string]interface{}{
		"to":      to,
		"subject": subject,
		"body":    body,
	})
}

// DeleteMailMessage deletes an email message.
func (c *Client) DeleteMailMessage(ctx context.Context, messageID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "delete-mail-message", map[string]interface{}{
		"messageId": messageID,
	})
}

// ===== Calendar Tools =====

// ListCalendars lists all calendars.
func (c *Client) ListCalendars(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-calendars", map[string]interface{}{})
}

// ListCalendarEvents lists calendar events.
func (c *Client) ListCalendarEvents(ctx context.Context, calendarID string, limit int) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if calendarID != "" {
		args["calendarId"] = calendarID
	}
	if limit > 0 {
		args["top"] = limit
	}
	return c.CallTool(ctx, "list-calendar-events", args)
}

// GetCalendarEvent retrieves a specific calendar event.
func (c *Client) GetCalendarEvent(ctx context.Context, eventID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "get-calendar-event", map[string]interface{}{
		"eventId": eventID,
	})
}

// GetCalendarView gets calendar view for a date range.
func (c *Client) GetCalendarView(ctx context.Context, startDateTime, endDateTime string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "get-calendar-view", map[string]interface{}{
		"startDateTime": startDateTime,
		"endDateTime":   endDateTime,
	})
}

// CreateCalendarEvent creates a new calendar event.
func (c *Client) CreateCalendarEvent(ctx context.Context, subject, start, end string, attendees []string) (*client.ToolResult, error) {
	args := map[string]interface{}{
		"subject":       subject,
		"startDateTime": start,
		"endDateTime":   end,
	}
	if len(attendees) > 0 {
		args["attendees"] = attendees
	}
	return c.CallTool(ctx, "create-calendar-event", args)
}

// DeleteCalendarEvent deletes a calendar event.
func (c *Client) DeleteCalendarEvent(ctx context.Context, eventID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "delete-calendar-event", map[string]interface{}{
		"eventId": eventID,
	})
}

// ===== OneDrive Tools =====

// ListDrives lists available drives.
func (c *Client) ListDrives(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-drives", map[string]interface{}{})
}

// GetDriveRootItem gets the root folder of a drive.
func (c *Client) GetDriveRootItem(ctx context.Context, driveID string) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if driveID != "" {
		args["driveId"] = driveID
	}
	return c.CallTool(ctx, "get-drive-root-item", args)
}

// ListFolderFiles lists files in a folder.
func (c *Client) ListFolderFiles(ctx context.Context, folderPath string) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if folderPath != "" {
		args["folderPath"] = folderPath
	}
	return c.CallTool(ctx, "list-folder-files", args)
}

// DownloadFile downloads a file from OneDrive.
func (c *Client) DownloadFile(ctx context.Context, filePath string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "download-onedrive-file-content", map[string]interface{}{
		"filePath": filePath,
	})
}

// UploadFile uploads content to an existing file.
func (c *Client) UploadFile(ctx context.Context, filePath, content string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "upload-file-content", map[string]interface{}{
		"filePath": filePath,
		"content":  content,
	})
}

// UploadNewFile uploads a new file.
func (c *Client) UploadNewFile(ctx context.Context, filePath, content string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "upload-new-file", map[string]interface{}{
		"filePath": filePath,
		"content":  content,
	})
}

// DeleteFile deletes a file from OneDrive.
func (c *Client) DeleteFile(ctx context.Context, filePath string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "delete-onedrive-file", map[string]interface{}{
		"filePath": filePath,
	})
}

// ===== To-Do Tasks Tools =====

// ListTodoTaskLists lists all task lists.
func (c *Client) ListTodoTaskLists(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-todo-task-lists", map[string]interface{}{})
}

// ListTodoTasks lists tasks in a task list.
func (c *Client) ListTodoTasks(ctx context.Context, taskListID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-todo-tasks", map[string]interface{}{
		"taskListId": taskListID,
	})
}

// CreateTodoTask creates a new task.
func (c *Client) CreateTodoTask(ctx context.Context, taskListID, title string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "create-todo-task", map[string]interface{}{
		"taskListId": taskListID,
		"title":      title,
	})
}

// UpdateTodoTask updates a task.
func (c *Client) UpdateTodoTask(ctx context.Context, taskListID, taskID string, completed bool) (*client.ToolResult, error) {
	status := "notStarted"
	if completed {
		status = "completed"
	}
	return c.CallTool(ctx, "update-todo-task", map[string]interface{}{
		"taskListId": taskListID,
		"taskId":     taskID,
		"status":     status,
	})
}

// DeleteTodoTask deletes a task.
func (c *Client) DeleteTodoTask(ctx context.Context, taskListID, taskID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "delete-todo-task", map[string]interface{}{
		"taskListId": taskListID,
		"taskId":     taskID,
	})
}

// ===== Contacts Tools =====

// ListContacts lists Outlook contacts.
func (c *Client) ListContacts(ctx context.Context, limit int) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if limit > 0 {
		args["top"] = limit
	}
	return c.CallTool(ctx, "list-outlook-contacts", args)
}

// GetContact retrieves a specific contact.
func (c *Client) GetContact(ctx context.Context, contactID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "get-outlook-contact", map[string]interface{}{
		"contactId": contactID,
	})
}

// CreateContact creates a new contact.
func (c *Client) CreateContact(ctx context.Context, displayName, email string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "create-outlook-contact", map[string]interface{}{
		"displayName":    displayName,
		"emailAddresses": []string{email},
	})
}

// ===== User & Profile Tools =====

// GetCurrentUser gets the current user's profile.
func (c *Client) GetCurrentUser(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "get-current-user", map[string]interface{}{})
}

// Search performs a search query.
func (c *Client) Search(ctx context.Context, query string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "search-query", map[string]interface{}{
		"query": query,
	})
}

// ===== Teams Tools (requires OrgMode) =====

// ListChats lists user's chats.
func (c *Client) ListChats(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-chats", map[string]interface{}{})
}

// ListJoinedTeams lists teams the user has joined.
func (c *Client) ListJoinedTeams(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "list-joined-teams", map[string]interface{}{})
}

// SendChatMessage sends a message to a chat.
func (c *Client) SendChatMessage(ctx context.Context, chatID, message string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "send-chat-message", map[string]interface{}{
		"chatId":  chatID,
		"message": message,
	})
}

// ===== Auth Tools =====

// Login initiates the device code login flow.
func (c *Client) Login(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "login", map[string]interface{}{})
}

// VerifyLogin verifies the login status.
func (c *Client) VerifyLogin(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "verify-login", map[string]interface{}{})
}
