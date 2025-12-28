// Package notion provides an MCP server for Notion integration.
package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
)

// NotionAPI defines the interface for Notion API operations
type NotionAPI interface {
	Get(ctx context.Context, path string) (map[string]interface{}, error)
	Post(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error)
	Patch(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error)
}

// Client is a Notion API client
type Client struct {
	token      string
	httpClient *http.Client
	baseURL    string
	version    string
}

// NewClient creates a new Notion client
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.notion.com/v1",
		version:    "2022-06-28",
	}
}

// Server wraps the MCP server with Notion functionality
type Server struct {
	*server.Server
	client NotionAPI
}

// New creates a new Notion MCP server
func New(client *Client) *Server {
	return newServer(client)
}

// NewWithMockClient creates a Notion MCP server with a mock client for testing
func NewWithMockClient(client NotionAPI) *Server {
	return newServer(client)
}

func newServer(client NotionAPI) *Server {
	s := &Server{
		Server: server.New(server.Config{Name: "notion", Version: "1.0.0"}),
		client: client,
	}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	// Search
	s.RegisterTool(
		server.NewTool("notion.search").
			Description("Search for pages and databases in Notion").
			String("query", "Search query", true).
			Enum("filter", "Filter by type", []string{"page", "database"}, false).
			Integer("limit", "Max results (default 10)", false).
			Build(),
		s.handleSearch,
	)

	// Get page
	s.RegisterTool(
		server.NewTool("notion.get_page").
			Description("Get a Notion page by ID").
			String("page_id", "Page ID", true).
			Build(),
		s.handleGetPage,
	)

	// Get page content
	s.RegisterTool(
		server.NewTool("notion.get_content").
			Description("Get the content blocks of a Notion page").
			String("page_id", "Page ID", true).
			Integer("limit", "Max blocks to return (default 50)", false).
			Build(),
		s.handleGetContent,
	)

	// Create page
	s.RegisterTool(
		server.NewTool("notion.create_page").
			Description("Create a new Notion page").
			String("parent_id", "Parent page or database ID", true).
			String("title", "Page title", true).
			String("content", "Page content in markdown (optional)", false).
			Build(),
		s.handleCreatePage,
	)

	// Update page
	s.RegisterTool(
		server.NewTool("notion.update_page").
			Description("Update page properties").
			String("page_id", "Page ID", true).
			String("title", "New title (optional)", false).
			Boolean("archived", "Archive the page", false).
			Build(),
		s.handleUpdatePage,
	)

	// Query database
	s.RegisterTool(
		server.NewTool("notion.query_database").
			Description("Query a Notion database").
			String("database_id", "Database ID", true).
			String("filter_property", "Property name to filter by (optional)", false).
			String("filter_value", "Value to filter for (optional)", false).
			Integer("limit", "Max results (default 50)", false).
			Build(),
		s.handleQueryDatabase,
	)

	// List databases
	s.RegisterTool(
		server.NewTool("notion.list_databases").
			Description("List all accessible databases").
			Integer("limit", "Max databases (default 10)", false).
			Build(),
		s.handleListDatabases,
	)

	// Get database
	s.RegisterTool(
		server.NewTool("notion.get_database").
			Description("Get database schema and properties").
			String("database_id", "Database ID", true).
			Build(),
		s.handleGetDatabase,
	)

	// Add comment
	s.RegisterTool(
		server.NewTool("notion.add_comment").
			Description("Add a comment to a page").
			String("page_id", "Page ID", true).
			String("text", "Comment text", true).
			Build(),
		s.handleAddComment,
	)

	// Get comments
	s.RegisterTool(
		server.NewTool("notion.get_comments").
			Description("Get comments on a page").
			String("page_id", "Page ID", true).
			Build(),
		s.handleGetComments,
	)
}

// handleSearch searches Notion
func (s *Server) handleSearch(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	query, err := args.RequireString("query")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	filterType := args.String("filter")
	limit := args.IntDefault("limit", 10)

	body := map[string]interface{}{
		"query":     query,
		"page_size": limit,
	}

	if filterType != "" {
		body["filter"] = map[string]string{
			"value":    filterType,
			"property": "object",
		}
	}

	resp, err := s.client.Post(ctx, "/search", body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Search failed: %v", err)), nil
	}

	results, _ := resp["results"].([]interface{})
	var items []map[string]interface{}

	for _, r := range results {
		item := r.(map[string]interface{})
		objectType := item["object"].(string)

		entry := map[string]interface{}{
			"id":   item["id"],
			"type": objectType,
		}

		if objectType == "page" {
			if props, ok := item["properties"].(map[string]interface{}); ok {
				entry["title"] = extractTitle(props)
			}
			entry["url"] = item["url"]
		} else if objectType == "database" {
			if title, ok := item["title"].([]interface{}); ok && len(title) > 0 {
				if text, ok := title[0].(map[string]interface{}); ok {
					if pt, ok := text["plain_text"].(string); ok {
						entry["title"] = pt
					}
				}
			}
		}

		items = append(items, entry)
	}

	return server.JSONResult(map[string]interface{}{
		"results": items,
		"count":   len(items),
		"query":   query,
	})
}

// handleGetPage gets a page
func (s *Server) handleGetPage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	pageID, err := args.RequireString("page_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.Get(ctx, "/pages/"+pageID)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get page: %v", err)), nil
	}

	props, _ := resp["properties"].(map[string]interface{})
	title := extractTitle(props)

	return server.JSONResult(map[string]interface{}{
		"id":          resp["id"],
		"title":       title,
		"url":         resp["url"],
		"created_time": resp["created_time"],
		"last_edited": resp["last_edited_time"],
		"archived":    resp["archived"],
		"properties":  props,
	})
}

// handleGetContent gets page content
func (s *Server) handleGetContent(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	pageID, err := args.RequireString("page_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	limit := args.IntDefault("limit", 50)

	resp, err := s.client.Get(ctx, fmt.Sprintf("/blocks/%s/children?page_size=%d", pageID, limit))
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get content: %v", err)), nil
	}

	results, _ := resp["results"].([]interface{})
	var blocks []map[string]interface{}

	for _, r := range results {
		block := r.(map[string]interface{})
		blockType := block["type"].(string)

		entry := map[string]interface{}{
			"id":   block["id"],
			"type": blockType,
		}

		// Extract text content based on block type
		if content, ok := block[blockType].(map[string]interface{}); ok {
			if richText, ok := content["rich_text"].([]interface{}); ok {
				entry["text"] = extractPlainText(richText)
			}
			if url, ok := content["url"].(string); ok {
				entry["url"] = url
			}
		}

		blocks = append(blocks, entry)
	}

	return server.JSONResult(map[string]interface{}{
		"blocks":   blocks,
		"count":    len(blocks),
		"has_more": resp["has_more"],
	})
}

// handleCreatePage creates a page
func (s *Server) handleCreatePage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	parentID, err := args.RequireString("parent_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	title, err := args.RequireString("title")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	content := args.String("content")

	body := map[string]interface{}{
		"parent": map[string]string{
			"page_id": parentID,
		},
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]string{
							"content": title,
						},
					},
				},
			},
		},
	}

	// Add content if provided
	if content != "" {
		body["children"] = []map[string]interface{}{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]interface{}{
					"rich_text": []map[string]interface{}{
						{
							"type": "text",
							"text": map[string]string{
								"content": content,
							},
						},
					},
				},
			},
		}
	}

	resp, err := s.client.Post(ctx, "/pages", body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to create page: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"id":      resp["id"],
		"url":     resp["url"],
		"message": "Page created successfully",
	})
}

// handleUpdatePage updates a page
func (s *Server) handleUpdatePage(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	pageID, err := args.RequireString("page_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	title := args.String("title")
	archived := args.Bool("archived")

	body := make(map[string]interface{})

	if title != "" {
		body["properties"] = map[string]interface{}{
			"title": map[string]interface{}{
				"title": []map[string]interface{}{
					{
						"text": map[string]string{
							"content": title,
						},
					},
				},
			},
		}
	}

	if args.Has("archived") {
		body["archived"] = archived
	}

	if len(body) == 0 {
		return server.ErrorResult("No updates specified"), nil
	}

	resp, err := s.client.Patch(ctx, "/pages/"+pageID, body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to update page: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"id":      resp["id"],
		"message": "Page updated successfully",
	})
}

// handleQueryDatabase queries a database
func (s *Server) handleQueryDatabase(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	dbID, err := args.RequireString("database_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	filterProp := args.String("filter_property")
	filterVal := args.String("filter_value")
	limit := args.IntDefault("limit", 50)

	body := map[string]interface{}{
		"page_size": limit,
	}

	if filterProp != "" && filterVal != "" {
		body["filter"] = map[string]interface{}{
			"property": filterProp,
			"rich_text": map[string]string{
				"contains": filterVal,
			},
		}
	}

	resp, err := s.client.Post(ctx, "/databases/"+dbID+"/query", body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to query database: %v", err)), nil
	}

	results, _ := resp["results"].([]interface{})
	var items []map[string]interface{}

	for _, r := range results {
		item := r.(map[string]interface{})
		props, _ := item["properties"].(map[string]interface{})

		entry := map[string]interface{}{
			"id":    item["id"],
			"url":   item["url"],
			"title": extractTitle(props),
		}

		// Include all properties
		for name, prop := range props {
			if propMap, ok := prop.(map[string]interface{}); ok {
				entry[name] = extractPropertyValue(propMap)
			}
		}

		items = append(items, entry)
	}

	return server.JSONResult(map[string]interface{}{
		"results":  items,
		"count":    len(items),
		"has_more": resp["has_more"],
	})
}

// handleListDatabases lists databases
func (s *Server) handleListDatabases(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	limit := args.IntDefault("limit", 10)

	body := map[string]interface{}{
		"filter": map[string]string{
			"value":    "database",
			"property": "object",
		},
		"page_size": limit,
	}

	resp, err := s.client.Post(ctx, "/search", body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to list databases: %v", err)), nil
	}

	results, _ := resp["results"].([]interface{})
	var databases []map[string]interface{}

	for _, r := range results {
		db := r.(map[string]interface{})
		title := ""
		if titleArr, ok := db["title"].([]interface{}); ok && len(titleArr) > 0 {
			if t, ok := titleArr[0].(map[string]interface{}); ok {
				title, _ = t["plain_text"].(string)
			}
		}

		databases = append(databases, map[string]interface{}{
			"id":    db["id"],
			"title": title,
			"url":   db["url"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"databases": databases,
		"count":     len(databases),
	})
}

// handleGetDatabase gets database schema
func (s *Server) handleGetDatabase(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	dbID, err := args.RequireString("database_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.Get(ctx, "/databases/"+dbID)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get database: %v", err)), nil
	}

	title := ""
	if titleArr, ok := resp["title"].([]interface{}); ok && len(titleArr) > 0 {
		if t, ok := titleArr[0].(map[string]interface{}); ok {
			title, _ = t["plain_text"].(string)
		}
	}

	// Extract property definitions
	props, _ := resp["properties"].(map[string]interface{})
	var properties []map[string]interface{}
	for name, prop := range props {
		if propMap, ok := prop.(map[string]interface{}); ok {
			properties = append(properties, map[string]interface{}{
				"name": name,
				"type": propMap["type"],
			})
		}
	}

	return server.JSONResult(map[string]interface{}{
		"id":         resp["id"],
		"title":      title,
		"url":        resp["url"],
		"properties": properties,
	})
}

// handleAddComment adds a comment
func (s *Server) handleAddComment(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	pageID, err := args.RequireString("page_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	text, err := args.RequireString("text")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	body := map[string]interface{}{
		"parent": map[string]string{
			"page_id": pageID,
		},
		"rich_text": []map[string]interface{}{
			{
				"text": map[string]string{
					"content": text,
				},
			},
		},
	}

	resp, err := s.client.Post(ctx, "/comments", body)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to add comment: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"id":      resp["id"],
		"message": "Comment added successfully",
	})
}

// handleGetComments gets comments
func (s *Server) handleGetComments(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	args := server.ParseArgs(raw)
	pageID, err := args.RequireString("page_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	resp, err := s.client.Get(ctx, "/comments?block_id="+pageID)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to get comments: %v", err)), nil
	}

	results, _ := resp["results"].([]interface{})
	var comments []map[string]interface{}

	for _, r := range results {
		comment := r.(map[string]interface{})
		richText, _ := comment["rich_text"].([]interface{})

		comments = append(comments, map[string]interface{}{
			"id":           comment["id"],
			"text":         extractPlainText(richText),
			"created_time": comment["created_time"],
		})
	}

	return server.JSONResult(map[string]interface{}{
		"comments": comments,
		"count":    len(comments),
	})
}

// HTTP helper methods

// Get implements NotionAPI.Get
func (c *Client) Get(ctx context.Context, path string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

// Post implements NotionAPI.Post
func (c *Client) Post(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

// Patch implements NotionAPI.Patch
func (c *Client) Patch(ctx context.Context, path string, body map[string]interface{}) (map[string]interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	return c.doRequest(req)
}

func (c *Client) doRequest(req *http.Request) (map[string]interface{}, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", c.version)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		errMsg, _ := result["message"].(string)
		return nil, fmt.Errorf("notion API error: %s", errMsg)
	}

	return result, nil
}

// Helper functions

func extractTitle(props map[string]interface{}) string {
	// Try "title" property first
	if titleProp, ok := props["title"]; ok {
		if titleMap, ok := titleProp.(map[string]interface{}); ok {
			if titleArr, ok := titleMap["title"].([]interface{}); ok {
				return extractPlainText(titleArr)
			}
		}
	}

	// Try "Name" property
	if nameProp, ok := props["Name"]; ok {
		if nameMap, ok := nameProp.(map[string]interface{}); ok {
			if titleArr, ok := nameMap["title"].([]interface{}); ok {
				return extractPlainText(titleArr)
			}
		}
	}

	return ""
}

func extractPlainText(richText []interface{}) string {
	var text string
	for _, rt := range richText {
		if rtMap, ok := rt.(map[string]interface{}); ok {
			if pt, ok := rtMap["plain_text"].(string); ok {
				text += pt
			}
		}
	}
	return text
}

func extractPropertyValue(prop map[string]interface{}) interface{} {
	propType, _ := prop["type"].(string)

	switch propType {
	case "title", "rich_text":
		if arr, ok := prop[propType].([]interface{}); ok {
			return extractPlainText(arr)
		}
	case "number":
		return prop["number"]
	case "checkbox":
		return prop["checkbox"]
	case "select":
		if sel, ok := prop["select"].(map[string]interface{}); ok {
			return sel["name"]
		}
	case "multi_select":
		if arr, ok := prop["multi_select"].([]interface{}); ok {
			var names []string
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok {
						names = append(names, name)
					}
				}
			}
			return names
		}
	case "date":
		if d, ok := prop["date"].(map[string]interface{}); ok {
			return d["start"]
		}
	case "url":
		return prop["url"]
	case "email":
		return prop["email"]
	case "phone_number":
		return prop["phone_number"]
	}

	return nil
}
