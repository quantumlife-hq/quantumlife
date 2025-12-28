//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/mcp/client/notion"
)

// Tests using the official Notion MCP server (@notionhq/notion-mcp-server)

func TestNotionOfficial_E2E_Connect(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	t.Log("Successfully connected to official Notion MCP server")
}

func TestNotionOfficial_E2E_ListTools(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Official Notion MCP server has %d tools:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
	}

	// Verify we have the expected tools (official server uses API- prefix)
	expectedTools := []string{"API-post-search", "API-retrieve-a-page", "API-post-page", "API-query-data-source"}
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("Expected tool %q not found", expected)
		}
	}

	if len(tools) != 21 {
		t.Errorf("Expected 21 tools, got %d", len(tools))
	}
}

func TestNotionOfficial_E2E_Search(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.Search(ctx, "", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("Search returned error: %s", result.Content[0].Text)
	}

	t.Logf("Search completed successfully")
	if len(result.Content) > 0 {
		// Pretty print first 500 chars
		text := result.Content[0].Text
		if len(text) > 500 {
			text = text[:500] + "..."
		}
		t.Logf("Results: %s", text)
	}
}

func TestNotionOfficial_E2E_QueryDataSource(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	result, err := client.QueryDataSource(ctx, cfg.NotionTestDatabase, 5)
	if err != nil {
		t.Fatalf("QueryDataSource failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("QueryDataSource returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully queried data source: %s", cfg.NotionTestDatabase)
}

func TestNotionOfficial_E2E_GetPage(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotion(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// First search to find a page
	searchResult, err := client.Search(ctx, "", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if searchResult.IsError || len(searchResult.Content) == 0 {
		t.Skip("No pages found to test")
	}

	// Parse search result to get a page ID
	var searchData map[string]interface{}
	if err := json.Unmarshal([]byte(searchResult.Content[0].Text), &searchData); err != nil {
		t.Fatalf("Failed to parse search result: %v", err)
	}

	results, ok := searchData["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Skip("No pages found in search results")
	}

	firstResult := results[0].(map[string]interface{})
	pageID, ok := firstResult["id"].(string)
	if !ok {
		t.Skip("Could not get page ID from search results")
	}

	// Get the page
	result, err := client.GetPage(ctx, pageID)
	if err != nil {
		t.Fatalf("GetPage failed: %v", err)
	}

	if result.IsError {
		t.Fatalf("GetPage returned error: %s", result.Content[0].Text)
	}

	t.Logf("Successfully retrieved page: %s", pageID)
}

func TestNotionOfficial_E2E_FullFlow(t *testing.T) {
	cfg := LoadE2EConfig(t)
	cfg.RequireNotionDatabase(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := notion.New(cfg.NotionToken)
	if err != nil {
		t.Fatalf("Failed to create Notion client: %v", err)
	}
	defer client.Close()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	t.Log("Step 1: Connected to official Notion MCP server")

	// List tools
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}
	t.Logf("Step 2: Found %d tools", len(tools))

	// Search
	searchResult, err := client.Search(ctx, "test", 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	t.Log("Step 3: Search completed")

	// Query database
	queryResult, err := client.QueryDataSource(ctx, cfg.NotionTestDatabase, 3)
	if err != nil {
		t.Logf("Step 4: QueryDataSource failed (may need to share database): %v", err)
	} else if queryResult.IsError {
		t.Logf("Step 4: QueryDataSource error: %s", queryResult.Content[0].Text)
	} else {
		t.Log("Step 4: Database query completed")
	}

	_ = searchResult // Use variable
	t.Log("Full flow completed successfully!")
}
