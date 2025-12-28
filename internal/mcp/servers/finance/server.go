// Package finance provides an MCP server for Plaid finance integration.
package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlife/quantumlife/internal/finance"
	"github.com/quantumlife/quantumlife/internal/mcp/server"
	"github.com/quantumlife/quantumlife/internal/spaces"
)

// FinanceSpace defines the interface for finance operations used by the server.
// This interface allows for mocking in unit tests.
type FinanceSpace interface {
	IsConnected() bool
	GetAccounts() []finance.Account
	GetTotalBalance() float64
	GetNetWorth() (assets, liabilities, netWorth float64)
	GetTransactions(filter finance.TransactionFilter) []*finance.CategorizedTransaction
	GetSpendingSummary(period string) *finance.SpendingSummary
	GetRecurringTransactions() []*finance.RecurringTransaction
	GetInsights() []*finance.Insight
	GetConnections() []*finance.Connection
	SetBudget(category finance.Category, amount float64)
	GetBudgets() map[finance.Category]float64
	CreateLinkToken(ctx context.Context, userID string) (string, error)
	GetSyncStatus() spaces.SyncStatus
}

// Server wraps the MCP server with finance functionality
type Server struct {
	*server.Server
	space FinanceSpace
}

// New creates a new Finance MCP server
func New(space *finance.Space) *Server {
	if space == nil {
		return newServer(nil)
	}
	return newServer(space)
}

// NewWithMockSpace creates a new Finance MCP server with a mock space for testing.
func NewWithMockSpace(space FinanceSpace) *Server {
	return newServer(space)
}

// newServer creates a new Finance MCP server with the given space.
func newServer(space FinanceSpace) *Server {
	s := &Server{
		Server: server.New(server.Config{Name: "finance", Version: "1.0.0"}),
		space:  space,
	}
	s.registerTools()
	s.registerResources()
	return s
}

func (s *Server) registerTools() {
	// List accounts
	s.RegisterTool(
		server.NewTool("finance.list_accounts").
			Description("List all connected bank accounts with balances").
			Build(),
		s.handleListAccounts,
	)

	// Get account balance
	s.RegisterTool(
		server.NewTool("finance.get_balance").
			Description("Get total balance and net worth across all accounts").
			Build(),
		s.handleGetBalance,
	)

	// List transactions
	s.RegisterTool(
		server.NewTool("finance.list_transactions").
			Description("List transactions with optional filtering").
			String("category", "Filter by category (food, transport, utilities, etc.)", false).
			String("start_date", "Start date (YYYY-MM-DD)", false).
			String("end_date", "End date (YYYY-MM-DD)", false).
			Number("min_amount", "Minimum transaction amount", false).
			Number("max_amount", "Maximum transaction amount", false).
			String("account_id", "Filter by account ID", false).
			Boolean("recurring_only", "Only show recurring transactions", false).
			Integer("limit", "Max transactions to return (default 50)", false).
			Build(),
		s.handleListTransactions,
	)

	// Get spending summary
	s.RegisterTool(
		server.NewTool("finance.spending_summary").
			Description("Get spending summary by category").
			Enum("period", "Time period", []string{"week", "month", "quarter", "year"}, false).
			Build(),
		s.handleSpendingSummary,
	)

	// Get recurring transactions
	s.RegisterTool(
		server.NewTool("finance.recurring").
			Description("Get detected recurring transactions (subscriptions, bills)").
			Build(),
		s.handleGetRecurring,
	)

	// Get insights
	s.RegisterTool(
		server.NewTool("finance.insights").
			Description("Get financial insights and recommendations").
			Build(),
		s.handleGetInsights,
	)

	// Get connections
	s.RegisterTool(
		server.NewTool("finance.connections").
			Description("List all bank connections").
			Build(),
		s.handleGetConnections,
	)

	// Set budget
	s.RegisterTool(
		server.NewTool("finance.set_budget").
			Description("Set a monthly budget for a category").
			String("category", "Spending category", true).
			Number("amount", "Monthly budget amount", true).
			Build(),
		s.handleSetBudget,
	)

	// Get budgets
	s.RegisterTool(
		server.NewTool("finance.get_budgets").
			Description("Get all configured budgets").
			Build(),
		s.handleGetBudgets,
	)

	// Create link token (for connecting new accounts)
	s.RegisterTool(
		server.NewTool("finance.create_link_token").
			Description("Create a Plaid Link token to connect a new bank account").
			String("user_id", "User identifier", true).
			Build(),
		s.handleCreateLinkToken,
	)

	// Search transactions
	s.RegisterTool(
		server.NewTool("finance.search").
			Description("Search transactions by merchant or description").
			String("query", "Search query", true).
			Integer("limit", "Max results (default 20)", false).
			Build(),
		s.handleSearchTransactions,
	)
}

func (s *Server) registerResources() {
	// Summary resource
	s.RegisterResource(
		server.Resource{
			URI:         "finance://summary",
			Name:        "Financial Summary",
			Description: "Overview of accounts, balances, and recent activity",
			MimeType:    "application/json",
		},
		s.handleSummaryResource,
	)

	// Monthly report resource
	s.RegisterResource(
		server.Resource{
			URI:         "finance://monthly",
			Name:        "Monthly Report",
			Description: "Spending breakdown for the current month",
			MimeType:    "application/json",
		},
		s.handleMonthlyResource,
	)
}

// Tool handlers

func (s *Server) handleListAccounts(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	accounts := s.space.GetAccounts()
	if len(accounts) == 0 {
		return server.SuccessResult("No accounts connected yet."), nil
	}

	var result []map[string]interface{}
	for _, acc := range accounts {
		result = append(result, map[string]interface{}{
			"id":        acc.AccountID,
			"name":      acc.Name,
			"type":      acc.Type,
			"subtype":   acc.Subtype,
			"mask":      acc.Mask,
			"balance":   acc.Balances.Current,
			"available": acc.Balances.Available,
			"currency":  acc.Balances.IsoCurrencyCode,
		})
	}

	return server.JSONResult(map[string]interface{}{
		"accounts": result,
		"count":    len(result),
	})
}

func (s *Server) handleGetBalance(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	assets, liabilities, netWorth := s.space.GetNetWorth()
	totalBalance := s.space.GetTotalBalance()

	return server.JSONResult(map[string]interface{}{
		"total_balance": totalBalance,
		"assets":        assets,
		"liabilities":   liabilities,
		"net_worth":     netWorth,
		"updated_at":    time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleListTransactions(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	args := server.ParseArgs(raw)
	category := args.String("category")
	startDate := args.String("start_date")
	endDate := args.String("end_date")
	minAmount := args.Float("min_amount")
	maxAmount := args.Float("max_amount")
	accountID := args.String("account_id")
	recurringOnly := args.Bool("recurring_only")
	limit := args.IntDefault("limit", 50)

	filter := finance.TransactionFilter{
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		AccountID: accountID,
	}

	if category != "" {
		filter.Categories = []finance.Category{finance.Category(category)}
	}
	if startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = t
		}
	}
	if endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			filter.EndDate = t
		}
	}
	if recurringOnly {
		recurring := true
		filter.Recurring = &recurring
	}

	transactions := s.space.GetTransactions(filter)

	// Apply limit
	if len(transactions) > limit {
		transactions = transactions[:limit]
	}

	var result []map[string]interface{}
	for _, tx := range transactions {
		result = append(result, map[string]interface{}{
			"id":           tx.TransactionID,
			"date":         tx.Date,
			"amount":       tx.Amount,
			"name":         tx.Name,
			"merchant":     tx.MerchantName,
			"category":     tx.QLCategory,
			"account_id":   tx.AccountID,
			"pending":      tx.Pending,
			"is_recurring": tx.IsRecurring,
		})
	}

	return server.JSONResult(map[string]interface{}{
		"transactions": result,
		"count":        len(result),
	})
}

func (s *Server) handleSpendingSummary(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	args := server.ParseArgs(raw)
	period := args.StringDefault("period", "month")

	summary := s.space.GetSpendingSummary(period)
	if summary == nil {
		return server.ErrorResult("No spending data available"), nil
	}

	return server.JSONResult(summary)
}

func (s *Server) handleGetRecurring(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	recurring := s.space.GetRecurringTransactions()

	var result []map[string]interface{}
	for _, r := range recurring {
		result = append(result, map[string]interface{}{
			"merchant":         r.MerchantName,
			"amount":           r.Amount,
			"frequency":        r.Frequency,
			"category":         r.Category,
			"next_expected":    r.NextExpected.Format("2006-01-02"),
			"last_seen":        r.LastSeen.Format("2006-01-02"),
			"occurrence_count": len(r.Transactions),
			"is_active":        r.IsActive,
		})
	}

	return server.JSONResult(map[string]interface{}{
		"recurring": result,
		"count":     len(result),
	})
}

func (s *Server) handleGetInsights(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	insights := s.space.GetInsights()

	var result []map[string]interface{}
	for _, i := range insights {
		result = append(result, map[string]interface{}{
			"type":        i.Type,
			"severity":    i.Severity,
			"title":       i.Title,
			"description": i.Description,
			"category":    i.Category,
			"amount":      i.Amount,
		})
	}

	return server.JSONResult(map[string]interface{}{
		"insights": result,
		"count":    len(result),
	})
}

func (s *Server) handleGetConnections(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil {
		return server.ErrorResult("Finance not configured"), nil
	}

	connections := s.space.GetConnections()

	var result []map[string]interface{}
	for _, c := range connections {
		result = append(result, map[string]interface{}{
			"id":          c.ID,
			"institution": c.InstitutionName,
			"status":      c.Status,
			"accounts":    len(c.Accounts),
			"last_sync":   c.LastSync.Format(time.RFC3339),
			"created_at":  c.CreatedAt.Format(time.RFC3339),
		})
	}

	return server.JSONResult(map[string]interface{}{
		"connections": result,
		"count":       len(result),
	})
}

func (s *Server) handleSetBudget(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil {
		return server.ErrorResult("Finance not configured"), nil
	}

	args := server.ParseArgs(raw)
	category, err := args.RequireString("category")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	amount := args.Float("amount")
	if amount <= 0 {
		return server.ErrorResult("Amount must be positive"), nil
	}

	s.space.SetBudget(finance.Category(category), amount)

	return server.SuccessResult(fmt.Sprintf("Budget set: $%.2f/month for %s", amount, category)), nil
}

func (s *Server) handleGetBudgets(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil {
		return server.ErrorResult("Finance not configured"), nil
	}

	budgets := s.space.GetBudgets()

	var result []map[string]interface{}
	for cat, amount := range budgets {
		result = append(result, map[string]interface{}{
			"category": string(cat),
			"budget":   amount,
		})
	}

	return server.JSONResult(map[string]interface{}{
		"budgets": result,
		"count":   len(result),
	})
}

func (s *Server) handleCreateLinkToken(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil {
		return server.ErrorResult("Finance not configured"), nil
	}

	args := server.ParseArgs(raw)
	userID, err := args.RequireString("user_id")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}

	linkToken, err := s.space.CreateLinkToken(ctx, userID)
	if err != nil {
		return server.ErrorResult(fmt.Sprintf("Failed to create link token: %v", err)), nil
	}

	return server.JSONResult(map[string]interface{}{
		"link_token": linkToken,
		"message":    "Use this token with Plaid Link to connect a bank account",
	})
}

func (s *Server) handleSearchTransactions(ctx context.Context, raw json.RawMessage) (*server.ToolResult, error) {
	if s.space == nil || !s.space.IsConnected() {
		return server.ErrorResult("Finance not connected. Connect a bank account first."), nil
	}

	args := server.ParseArgs(raw)
	query, err := args.RequireString("query")
	if err != nil {
		return server.ErrorResult(err.Error()), nil
	}
	limit := args.IntDefault("limit", 20)

	// Get all transactions and filter by query
	transactions := s.space.GetTransactions(finance.TransactionFilter{})

	var matches []map[string]interface{}
	queryLower := stringToLower(query)
	for _, tx := range transactions {
		if len(matches) >= limit {
			break
		}
		// Search in name and merchant
		if containsIgnoreCase(tx.Name, queryLower) || containsIgnoreCase(tx.MerchantName, queryLower) {
			matches = append(matches, map[string]interface{}{
				"id":       tx.TransactionID,
				"date":     tx.Date,
				"amount":   tx.Amount,
				"name":     tx.Name,
				"merchant": tx.MerchantName,
				"category": tx.QLCategory,
			})
		}
	}

	return server.JSONResult(map[string]interface{}{
		"matches": matches,
		"count":   len(matches),
		"query":   query,
	})
}

// Resource handlers

func (s *Server) handleSummaryResource(ctx context.Context, uri string) (*server.ResourceContent, error) {
	if s.space == nil || !s.space.IsConnected() {
		return &server.ResourceContent{
			URI:      uri,
			MimeType: "application/json",
			Text:     `{"status": "not_connected", "message": "Connect a bank account to see financial summary"}`,
		}, nil
	}

	accounts := s.space.GetAccounts()
	assets, liabilities, netWorth := s.space.GetNetWorth()
	connections := s.space.GetConnections()
	recurring := s.space.GetRecurringTransactions()

	summary := map[string]interface{}{
		"status": "connected",
		"accounts": map[string]interface{}{
			"count":         len(accounts),
			"total_balance": s.space.GetTotalBalance(),
		},
		"net_worth": map[string]interface{}{
			"assets":      assets,
			"liabilities": liabilities,
			"total":       netWorth,
		},
		"connections": len(connections),
		"recurring_transactions": len(recurring),
		"last_sync":              s.space.GetSyncStatus().LastSync.Format(time.RFC3339),
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	return &server.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

func (s *Server) handleMonthlyResource(ctx context.Context, uri string) (*server.ResourceContent, error) {
	if s.space == nil || !s.space.IsConnected() {
		return &server.ResourceContent{
			URI:      uri,
			MimeType: "application/json",
			Text:     `{"status": "not_connected"}`,
		}, nil
	}

	summary := s.space.GetSpendingSummary("month")
	if summary == nil {
		return &server.ResourceContent{
			URI:      uri,
			MimeType: "application/json",
			Text:     `{"status": "no_data"}`,
		}, nil
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	return &server.ResourceContent{
		URI:      uri,
		MimeType: "application/json",
		Text:     string(data),
	}, nil
}

// Helper functions

func stringToLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

func containsIgnoreCase(s, substr string) bool {
	sLower := stringToLower(s)
	return len(sLower) >= len(substr) && indexOfString(sLower, substr) >= 0
}

func indexOfString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
