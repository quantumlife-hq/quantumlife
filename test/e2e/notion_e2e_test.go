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

func TestNotion_E2E_GetContent(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First search to find a page with content
	searchArgs := json.RawMessage(`{"query": "test", "filter": "page", "limit": 1}`)
	searchResult, err := callNotionTool(srv, ctx, "notion.search", searchArgs)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if searchResult.IsError {
		t.Skipf("Search failed: %s", searchResult.Content[0].Text)
	}

	// Parse search results to get a page ID
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(searchResult.Content[0].Text), &searchData); err != nil {
		t.Fatalf("failed to parse search results: %v", err)
	}

	results, ok := searchData["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Skip("No pages found to test get_content")
	}

	firstResult := results[0].(map[string]interface{})
	pageID := firstResult["id"].(string)

	// Get the page content
	args, _ := json.Marshal(map[string]interface{}{
		"page_id": pageID,
		"limit":   10,
	})

	result, err := callNotionTool(srv, ctx, "notion.get_content", args)
	if err != nil {
		t.Fatalf("get_content failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_content returned error: %s", result.Content[0].Text)
	}

	// Verify response structure
	var contentData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &contentData); err != nil {
		t.Fatalf("failed to parse content data: %v", err)
	}

	if _, ok := contentData["blocks"]; !ok {
		t.Error("expected blocks field in response")
	}

	t.Logf("Successfully fetched content for page: %s (blocks: %v)", pageID, contentData["count"])
}

func TestNotion_E2E_GetDatabase(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args, _ := json.Marshal(map[string]string{
		"database_id": cfg.NotionTestDatabase,
	})

	result, err := callNotionTool(srv, ctx, "notion.get_database", args)
	if err != nil {
		t.Fatalf("get_database failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("get_database returned error: %s", result.Content[0].Text)
	}

	// Verify response structure
	var dbData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &dbData); err != nil {
		t.Fatalf("failed to parse database data: %v", err)
	}

	if dbData["id"] == nil {
		t.Error("expected id field in response")
	}
	if dbData["properties"] == nil {
		t.Error("expected properties field in response")
	}

	t.Logf("Successfully fetched database schema: %v (title: %v)", dbData["id"], dbData["title"])
}

func TestNotion_E2E_SearchFiltered(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test filtered search for pages only
	args := json.RawMessage(`{"query": "test", "filter": "page", "limit": 3}`)
	result, err := callNotionTool(srv, ctx, "notion.search", args)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("search returned error: %s", result.Content[0].Text)
	}

	// Verify all results are pages
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &searchData); err != nil {
		t.Fatalf("failed to parse search results: %v", err)
	}

	results, ok := searchData["results"].([]interface{})
	if ok {
		for _, r := range results {
			item := r.(map[string]interface{})
			if item["type"] != "page" {
				t.Errorf("expected type 'page', got %v", item["type"])
			}
		}
	}

	t.Logf("Successfully searched with page filter (found %v)", searchData["count"])
}

func TestNotion_E2E_SearchDatabaseFilter(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Test filtered search for databases only
	args := json.RawMessage(`{"query": "test", "filter": "database", "limit": 3}`)
	result, err := callNotionTool(srv, ctx, "notion.search", args)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("search returned error: %s", result.Content[0].Text)
	}

	// Verify all results are databases
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &searchData); err != nil {
		t.Fatalf("failed to parse search results: %v", err)
	}

	results, ok := searchData["results"].([]interface{})
	if ok {
		for _, r := range results {
			item := r.(map[string]interface{})
			if item["type"] != "database" {
				t.Errorf("expected type 'database', got %v", item["type"])
			}
		}
	}

	t.Logf("Successfully searched with database filter (found %v)", searchData["count"])
}

func TestNotion_E2E_UpdatePageTitle(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test page
	originalTitle := "E2E Test Page - " + time.Now().Format(time.RFC3339)
	createArgs, _ := json.Marshal(map[string]string{
		"parent_id": cfg.NotionTestDatabase,
		"title":     originalTitle,
	})

	createResult, err := callNotionTool(srv, ctx, "notion.create_page", createArgs)
	if err != nil {
		t.Fatalf("create_page failed: %v", err)
	}

	if createResult.IsError {
		t.Fatalf("create_page returned error: %s", createResult.Content[0].Text)
	}

	var pageData map[string]interface{}
	if err := json.Unmarshal([]byte(createResult.Content[0].Text), &pageData); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}

	pageID := pageData["id"].(string)
	t.Logf("Created page: %s", pageID)

	// Update the title
	newTitle := "Updated Title - " + time.Now().Format(time.RFC3339)
	updateArgs, _ := json.Marshal(map[string]interface{}{
		"page_id": pageID,
		"title":   newTitle,
	})

	updateResult, err := callNotionTool(srv, ctx, "notion.update_page", updateArgs)
	if err != nil {
		t.Fatalf("update_page failed: %v", err)
	}

	if updateResult.IsError {
		t.Fatalf("update_page returned error: %s", updateResult.Content[0].Text)
	}

	t.Logf("Successfully updated page title")

	// Cleanup - archive the page
	archiveArgs, _ := json.Marshal(map[string]interface{}{
		"page_id":  pageID,
		"archived": true,
	})
	callNotionTool(srv, ctx, "notion.update_page", archiveArgs)
}

func TestNotion_E2E_QueryDatabaseWithFilter(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First, query without filter to see what properties exist
	args, _ := json.Marshal(map[string]interface{}{
		"database_id": cfg.NotionTestDatabase,
		"limit":       3,
	})

	result, err := callNotionTool(srv, ctx, "notion.query_database", args)
	if err != nil {
		t.Fatalf("query_database failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("query_database returned error: %s", result.Content[0].Text)
	}

	// Verify response structure
	var queryData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &queryData); err != nil {
		t.Fatalf("failed to parse query data: %v", err)
	}

	if _, ok := queryData["results"]; !ok {
		t.Error("expected results field in response")
	}
	if _, ok := queryData["count"]; !ok {
		t.Error("expected count field in response")
	}

	t.Logf("Successfully queried database with results: %v", queryData["count"])
}

func TestNotion_E2E_CommentsLifecycle(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	srv := createNotionServer(t, cfg.NotionToken)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create a test page first
	testTitle := "E2E Comments Test - " + time.Now().Format(time.RFC3339)
	createArgs, _ := json.Marshal(map[string]string{
		"parent_id": cfg.NotionTestDatabase,
		"title":     testTitle,
	})

	createResult, err := callNotionTool(srv, ctx, "notion.create_page", createArgs)
	if err != nil {
		t.Fatalf("create_page failed: %v", err)
	}

	if createResult.IsError {
		t.Fatalf("create_page returned error: %s", createResult.Content[0].Text)
	}

	var pageData map[string]interface{}
	json.Unmarshal([]byte(createResult.Content[0].Text), &pageData)
	pageID := pageData["id"].(string)

	t.Logf("Created test page: %s", pageID)

	// Add a comment
	commentText := "E2E test comment - " + time.Now().Format(time.RFC3339)
	addCommentArgs, _ := json.Marshal(map[string]string{
		"page_id": pageID,
		"text":    commentText,
	})

	addResult, err := callNotionTool(srv, ctx, "notion.add_comment", addCommentArgs)
	if err != nil {
		t.Fatalf("add_comment failed: %v", err)
	}

	// Note: Comments API requires specific permissions, may fail
	if addResult.IsError {
		t.Logf("add_comment returned error (may need permissions): %s", addResult.Content[0].Text)
	} else {
		t.Logf("Successfully added comment")

		// Get comments
		getCommentsArgs, _ := json.Marshal(map[string]string{
			"page_id": pageID,
		})

		getResult, err := callNotionTool(srv, ctx, "notion.get_comments", getCommentsArgs)
		if err != nil {
			t.Fatalf("get_comments failed: %v", err)
		}

		if getResult.IsError {
			t.Logf("get_comments returned error: %s", getResult.Content[0].Text)
		} else {
			var commentsData map[string]interface{}
			json.Unmarshal([]byte(getResult.Content[0].Text), &commentsData)
			t.Logf("Successfully retrieved comments: %v", commentsData["count"])
		}
	}

	// Cleanup - archive the page
	archiveArgs, _ := json.Marshal(map[string]interface{}{
		"page_id":  pageID,
		"archived": true,
	})
	callNotionTool(srv, ctx, "notion.update_page", archiveArgs)
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
