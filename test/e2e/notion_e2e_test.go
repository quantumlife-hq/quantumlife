//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/server"
	"github.com/quantumlife/quantumlife/internal/mcp/servers/notion"
)

func TestNotion_E2E_Search(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := json.RawMessage(`{"query": "test", "limit": 5}`)
	result, err := callNotionTool(srv, ctx, "notion.search", args)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("search returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully searched Notion")
}

func TestNotion_E2E_GetPage(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First search to find a page
	searchArgs := json.RawMessage(`{"query": "", "limit": 1}`)
	searchResult, err := callNotionTool(srv, ctx, "notion.search", searchArgs)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if searchResult.IsError {
		t.Skipf("No pages found to test get_page")
	}

	// Parse search results to get a page ID
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(searchResult.Content[0].Text), &searchData); err != nil {
		t.Fatalf("failed to parse search results: %v", err)
	}

	results, ok := searchData["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Skip("No pages found to test get_page")
	}

	firstResult := results[0].(map[string]interface{})
	pageID := firstResult["id"].(string)

	// Now get the page
	args, _ := json.Marshal(map[string]string{"page_id": pageID})
	result, err := callNotionTool(srv, ctx, "notion.get_page", args)
	if err != nil {
		t.Fatalf("get_page failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_page returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully fetched page: %s", pageID)
}

func TestNotion_E2E_QueryDatabase(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]interface{}{
		"database_id": cfg.NotionTestDatabase,
		"limit":       5,
	})

	result, err := callNotionTool(srv, ctx, "notion.query_database", args)
	if err != nil {
		t.Fatalf("query_database failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("query_database returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully queried database: %s", cfg.NotionTestDatabase)
}

func TestNotion_E2E_ListDatabases(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := json.RawMessage(`{"limit": 5}`)
	result, err := callNotionTool(srv, ctx, "notion.list_databases", args)
	if err != nil {
		t.Fatalf("list_databases failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("list_databases returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully listed databases")
}

func TestNotion_E2E_CreateAndArchivePage(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test page
	testTitle := "E2E Test Page - " + time.Now().Format(time.RFC3339)
	createArgs, _ := json.Marshal(map[string]string{
		"parent_id": cfg.NotionTestDatabase,
		"title":     testTitle,
		"content":   "This is a test page created by E2E tests.",
	})

	createResult, err := callNotionTool(srv, ctx, "notion.create_page", createArgs)
	if err != nil {
		t.Fatalf("create_page failed: %v", err)
	}

	if createResult.IsError {
		t.Fatalf("create_page returned error: %s", createResult.Content[0].Text)
	}

	// Parse the result to get page ID
	var pageData map[string]interface{}
	if err := json.Unmarshal([]byte(createResult.Content[0].Text), &pageData); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	pageID, ok := pageData["id"].(string)
	if !ok {
		t.Fatalf("no page ID in create result")
	}

	t.Logf("Successfully created page: %s", pageID)

	// Archive the page (cleanup)
	archiveArgs, _ := json.Marshal(map[string]interface{}{
		"page_id":  pageID,
		"archived": true,
	})

	archiveResult, err := callNotionTool(srv, ctx, "notion.update_page", archiveArgs)
	if err != nil {
		t.Fatalf("update_page (archive) failed: %v", err)
	}

	if archiveResult.IsError {
		t.Fatalf("update_page (archive) returned error: %s", archiveResult.Content[0].Text)
	}

	t.Logf("Successfully archived page: %s", pageID)
}

// Helper to create a Notion MCP server with real credentials
func createNotionServer(t *testing.T, token string) *notion.Server {
	t.Helper()
	client := notion.NewClient(token)
	return notion.New(client)
}

// Helper to call a tool on the Notion server
func callNotionTool(srv *notion.Server, ctx context.Context, toolName string, args json.RawMessage) (*server.ToolResult, error) {
	_, handler, ok := srv.Registry().GetTool(toolName)
	if !ok {
		return nil, nil
	}
	return handler(ctx, args)
}
