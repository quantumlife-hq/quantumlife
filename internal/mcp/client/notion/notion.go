// Package notion provides a client for the official Notion MCP server.
package notion

import (
	"context"
	"fmt"

	"github.com/quantumlife/quantumlife/internal/mcp/client"
)

// Client wraps the official Notion MCP server.
type Client struct {
	*client.Client
	token string
}

// New creates a new Notion MCP client using the official @notionhq/notion-mcp-server.
func New(token string) (*Client, error) {
	c, err := client.New("npx", []string{"-y", "@notionhq/notion-mcp-server"}, []string{
		"NOTION_TOKEN=" + token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start Notion MCP server: %w", err)
	}

	return &Client{
		Client: c,
		token:  token,
	}, nil
}

// Connect initializes the connection to the Notion MCP server.
func (c *Client) Connect(ctx context.Context) error {
	return c.Initialize(ctx)
}

// Search searches Notion pages and databases.
func (c *Client) Search(ctx context.Context, query string, limit int) (*client.ToolResult, error) {
	args := map[string]interface{}{}
	if query != "" {
		args["query"] = query
	}
	if limit > 0 {
		args["page_size"] = limit
	}
	return c.CallTool(ctx, "API-post-search", args)
}

// GetPage retrieves a page by ID.
func (c *Client) GetPage(ctx context.Context, pageID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-retrieve-a-page", map[string]interface{}{
		"page_id": pageID,
	})
}

// CreatePage creates a new page.
func (c *Client) CreatePage(ctx context.Context, parentID, title string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-post-page", map[string]interface{}{
		"parent": map[string]interface{}{
			"type":    "page_id",
			"page_id": parentID,
		},
		"properties": map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]string{
						"content": title,
					},
				},
			},
		},
	})
}

// UpdatePage updates a page.
func (c *Client) UpdatePage(ctx context.Context, pageID string, archived bool) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-patch-page", map[string]interface{}{
		"page_id":  pageID,
		"archived": archived,
	})
}

// QueryDataSource queries a database/data source.
func (c *Client) QueryDataSource(ctx context.Context, dataSourceID string, limit int) (*client.ToolResult, error) {
	args := map[string]interface{}{
		"data_source_id": dataSourceID,
	}
	if limit > 0 {
		args["page_size"] = limit
	}
	return c.CallTool(ctx, "API-query-data-source", args)
}

// GetBlockChildren retrieves child blocks of a block/page.
func (c *Client) GetBlockChildren(ctx context.Context, blockID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-get-block-children", map[string]interface{}{
		"block_id": blockID,
	})
}

// AppendBlockChildren appends blocks to a page.
func (c *Client) AppendBlockChildren(ctx context.Context, blockID string, children []map[string]interface{}) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-patch-block-children", map[string]interface{}{
		"block_id": blockID,
		"children": children,
	})
}

// CreateComment creates a comment on a page.
func (c *Client) CreateComment(ctx context.Context, pageID, text string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-create-a-comment", map[string]interface{}{
		"parent": map[string]interface{}{
			"page_id": pageID,
		},
		"rich_text": []map[string]interface{}{
			{
				"type": "text",
				"text": map[string]string{
					"content": text,
				},
			},
		},
	})
}

// GetComments retrieves comments on a block/page.
func (c *Client) GetComments(ctx context.Context, blockID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-retrieve-a-comment", map[string]interface{}{
		"block_id": blockID,
	})
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, userID string) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-get-user", map[string]interface{}{
		"user_id": userID,
	})
}

// GetSelf retrieves the bot user for the token.
func (c *Client) GetSelf(ctx context.Context) (*client.ToolResult, error) {
	return c.CallTool(ctx, "API-get-self", map[string]interface{}{})
}
