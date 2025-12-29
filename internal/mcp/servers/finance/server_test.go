package finance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/finance"
	"github.com/quantumlife/quantumlife/internal/spaces"
)

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNew(t *testing.T) {
	t.Run("nil space returns server with nil space", func(t *testing.T) {
		srv := New(nil)
		if srv == nil {
			t.Fatal("New(nil) returned nil server")
		}
		if srv.space != nil {
			t.Error("expected nil space")
		}
	})
}

func TestNewWithMockSpace(t *testing.T) {
	t.Run("creates server with mock space", func(t *testing.T) {
		mock := &MockFinanceSpace{}
		srv := NewWithMockSpace(mock)
		if srv == nil {
			t.Fatal("NewWithMockSpace returned nil")
		}
		if srv.Server == nil {
			t.Error("Server is nil")
		}
		if srv.space == nil {
			t.Error("space is nil")
		}

		info := srv.Info()
		if info.Name != "finance" {
			t.Errorf("expected name 'finance', got %q", info.Name)
		}
		if info.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %q", info.Version)
		}
	})
}

// MockFinanceSpace implements a mock Finance space for testing.
type MockFinanceSpace struct {
	IsConnectedFunc              func() bool
	GetAccountsFunc              func() []finance.Account
	GetTotalBalanceFunc          func() float64
	GetNetWorthFunc              func() (assets, liabilities, netWorth float64)
	GetTransactionsFunc          func(filter finance.TransactionFilter) []*finance.CategorizedTransaction
	GetSpendingSummaryFunc       func(period string) *finance.SpendingSummary
	GetRecurringTransactionsFunc func() []*finance.RecurringTransaction
	GetInsightsFunc              func() []*finance.Insight
	GetConnectionsFunc           func() []*finance.Connection
	SetBudgetFunc                func(category finance.Category, amount float64)
	GetBudgetsFunc               func() map[finance.Category]float64
	CreateLinkTokenFunc          func(ctx context.Context, userID string) (string, error)
	GetSyncStatusFunc            func() spaces.SyncStatus
}

func (m *MockFinanceSpace) IsConnected() bool {
	if m.IsConnectedFunc != nil {
		return m.IsConnectedFunc()
	}
	return true
}

func (m *MockFinanceSpace) GetAccounts() []finance.Account {
	if m.GetAccountsFunc != nil {
		return m.GetAccountsFunc()
	}
	return sampleAccounts()
}

func (m *MockFinanceSpace) GetTotalBalance() float64 {
	if m.GetTotalBalanceFunc != nil {
		return m.GetTotalBalanceFunc()
	}
	return 15000.00
}

func (m *MockFinanceSpace) GetNetWorth() (assets, liabilities, netWorth float64) {
	if m.GetNetWorthFunc != nil {
		return m.GetNetWorthFunc()
	}
	return 20000.00, 5000.00, 15000.00
}

func (m *MockFinanceSpace) GetTransactions(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
	if m.GetTransactionsFunc != nil {
		return m.GetTransactionsFunc(filter)
	}
	return sampleTransactions()
}

func (m *MockFinanceSpace) GetSpendingSummary(period string) *finance.SpendingSummary {
	if m.GetSpendingSummaryFunc != nil {
		return m.GetSpendingSummaryFunc(period)
	}
	return &finance.SpendingSummary{
		Period:      period,
		TotalSpent:  2500.00,
		TotalIncome: 5000.00,
		NetCashFlow: 2500.00,
		ByCategory: map[finance.Category]float64{
			finance.CategoryGroceries: 500.00,
			finance.CategoryDining:    300.00,
		},
	}
}

func (m *MockFinanceSpace) GetRecurringTransactions() []*finance.RecurringTransaction {
	if m.GetRecurringTransactionsFunc != nil {
		return m.GetRecurringTransactionsFunc()
	}
	return sampleRecurring()
}

func (m *MockFinanceSpace) GetInsights() []*finance.Insight {
	if m.GetInsightsFunc != nil {
		return m.GetInsightsFunc()
	}
	return sampleInsights()
}

func (m *MockFinanceSpace) GetConnections() []*finance.Connection {
	if m.GetConnectionsFunc != nil {
		return m.GetConnectionsFunc()
	}
	return sampleConnections()
}

func (m *MockFinanceSpace) SetBudget(category finance.Category, amount float64) {
	if m.SetBudgetFunc != nil {
		m.SetBudgetFunc(category, amount)
	}
}

func (m *MockFinanceSpace) GetBudgets() map[finance.Category]float64 {
	if m.GetBudgetsFunc != nil {
		return m.GetBudgetsFunc()
	}
	return map[finance.Category]float64{
		finance.CategoryGroceries: 600.00,
		finance.CategoryDining:    400.00,
	}
}

func (m *MockFinanceSpace) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	if m.CreateLinkTokenFunc != nil {
		return m.CreateLinkTokenFunc(ctx, userID)
	}
	return "link-sandbox-test-token", nil
}

func (m *MockFinanceSpace) GetSyncStatus() spaces.SyncStatus {
	if m.GetSyncStatusFunc != nil {
		return m.GetSyncStatusFunc()
	}
	return spaces.SyncStatus{
		Status:    "idle",
		LastSync:  time.Now(),
		ItemCount: 100,
	}
}

// Sample data helpers
func sampleAccounts() []finance.Account {
	return []finance.Account{
		{
			AccountID: "acc-001",
			Name:      "Checking Account",
			Type:      "depository",
			Subtype:   "checking",
			Mask:      "1234",
			Balances: finance.AccountBalance{
				Current:         5000.00,
				Available:       4800.00,
				IsoCurrencyCode: "USD",
			},
		},
		{
			AccountID: "acc-002",
			Name:      "Savings Account",
			Type:      "depository",
			Subtype:   "savings",
			Mask:      "5678",
			Balances: finance.AccountBalance{
				Current:         10000.00,
				Available:       10000.00,
				IsoCurrencyCode: "USD",
			},
		},
	}
}

func sampleTransactions() []*finance.CategorizedTransaction {
	return []*finance.CategorizedTransaction{
		{
			Transaction: finance.Transaction{
				TransactionID: "tx-001",
				AccountID:     "acc-001",
				Amount:        45.67,
				Date:          "2024-01-15",
				Name:          "Whole Foods Market",
				MerchantName:  "Whole Foods",
				Pending:       false,
			},
			QLCategory:  finance.CategoryGroceries,
			IsRecurring: false,
		},
		{
			Transaction: finance.Transaction{
				TransactionID: "tx-002",
				AccountID:     "acc-001",
				Amount:        15.99,
				Date:          "2024-01-14",
				Name:          "Netflix",
				MerchantName:  "Netflix",
				Pending:       false,
			},
			QLCategory:  finance.CategorySubscription,
			IsRecurring: true,
		},
	}
}

func sampleRecurring() []*finance.RecurringTransaction {
	return []*finance.RecurringTransaction{
		{
			ID:           "rec-001",
			MerchantName: "Netflix",
			Category:     finance.CategorySubscription,
			Amount:       15.99,
			Frequency:    "monthly",
			NextExpected: time.Now().AddDate(0, 1, 0),
			LastSeen:     time.Now(),
			IsActive:     true,
		},
		{
			ID:           "rec-002",
			MerchantName: "Spotify",
			Category:     finance.CategorySubscription,
			Amount:       9.99,
			Frequency:    "monthly",
			NextExpected: time.Now().AddDate(0, 1, 0),
			LastSeen:     time.Now(),
			IsActive:     true,
		},
	}
}

func sampleInsights() []*finance.Insight {
	return []*finance.Insight{
		{
			ID:          "insight-001",
			Type:        finance.InsightTypeBudgetAlert,
			Title:       "Dining budget exceeded",
			Description: "You've spent $450 on dining, exceeding your $400 budget",
			Severity:    finance.SeverityWarning,
			Amount:      50.00,
			Category:    finance.CategoryDining,
		},
	}
}

func sampleConnections() []*finance.Connection {
	return []*finance.Connection{
		{
			ID:              "conn-001",
			InstitutionID:   "ins_1",
			InstitutionName: "Chase Bank",
			Status:          "active",
			Accounts:        sampleAccounts(),
			LastSync:        time.Now(),
			CreatedAt:       time.Now().AddDate(0, -1, 0),
		},
	}
}

// Tests

func TestFinanceServer_ListAccounts(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "list accounts successfully",
			wantErr: false,
		},
		{
			name: "no accounts connected",
			setup: func(m *MockFinanceSpace) {
				m.GetAccountsFunc = func() []finance.Account {
					return []finance.Account{}
				}
			},
			wantErr: false,
		},
		{
			name: "not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleListAccounts(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_GetBalance(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get balance successfully",
			wantErr: false,
		},
		{
			name: "not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleGetBalance(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_ListTransactions(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "list transactions default",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "list transactions with category filter",
			args: map[string]interface{}{
				"category": "groceries",
			},
			setup: func(m *MockFinanceSpace) {
				m.GetTransactionsFunc = func(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
					if len(filter.Categories) == 0 || filter.Categories[0] != finance.CategoryGroceries {
						t.Error("expected groceries category filter")
					}
					return sampleTransactions()
				}
			},
			wantErr: false,
		},
		{
			name: "list transactions with date range",
			args: map[string]interface{}{
				"start_date": "2024-01-01",
				"end_date":   "2024-01-31",
			},
			wantErr: false,
		},
		{
			name: "list transactions with amount range",
			args: map[string]interface{}{
				"min_amount": 10.00,
				"max_amount": 100.00,
			},
			wantErr: false,
		},
		{
			name: "list transactions recurring only",
			args: map[string]interface{}{
				"recurring_only": true,
			},
			wantErr: false,
		},
		{
			name: "list transactions with limit",
			args: map[string]interface{}{
				"limit": 10,
			},
			wantErr: false,
		},
		{
			name: "not connected",
			args: map[string]interface{}{},
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleListTransactions(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_SpendingSummary(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get spending summary default",
			args:    map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "get spending summary for week",
			args: map[string]interface{}{
				"period": "week",
			},
			setup: func(m *MockFinanceSpace) {
				m.GetSpendingSummaryFunc = func(period string) *finance.SpendingSummary {
					if period != "week" {
						t.Errorf("expected period 'week', got '%s'", period)
					}
					return &finance.SpendingSummary{Period: period}
				}
			},
			wantErr: false,
		},
		{
			name: "get spending summary for quarter",
			args: map[string]interface{}{
				"period": "quarter",
			},
			wantErr: false,
		},
		{
			name: "no spending data",
			args: map[string]interface{}{},
			setup: func(m *MockFinanceSpace) {
				m.GetSpendingSummaryFunc = func(period string) *finance.SpendingSummary {
					return nil
				}
			},
			wantErr: true,
		},
		{
			name: "not connected",
			args: map[string]interface{}{},
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleSpendingSummary(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_GetRecurring(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get recurring transactions",
			wantErr: false,
		},
		{
			name: "no recurring transactions",
			setup: func(m *MockFinanceSpace) {
				m.GetRecurringTransactionsFunc = func() []*finance.RecurringTransaction {
					return []*finance.RecurringTransaction{}
				}
			},
			wantErr: false,
		},
		{
			name: "not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleGetRecurring(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_GetInsights(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get insights",
			wantErr: false,
		},
		{
			name: "no insights",
			setup: func(m *MockFinanceSpace) {
				m.GetInsightsFunc = func() []*finance.Insight {
					return []*finance.Insight{}
				}
			},
			wantErr: false,
		},
		{
			name: "not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleGetInsights(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_GetConnections(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get connections",
			wantErr: false,
		},
		{
			name: "no connections",
			setup: func(m *MockFinanceSpace) {
				m.GetConnectionsFunc = func() []*finance.Connection {
					return []*finance.Connection{}
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleGetConnections(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_SetBudget(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name: "set budget successfully",
			args: map[string]interface{}{
				"category": "groceries",
				"amount":   500.00,
			},
			setup: func(m *MockFinanceSpace) {
				m.SetBudgetFunc = func(category finance.Category, amount float64) {
					if category != finance.CategoryGroceries {
						t.Errorf("expected groceries category, got %s", category)
					}
					if amount != 500.00 {
						t.Errorf("expected amount 500, got %.2f", amount)
					}
				}
			},
			wantErr: false,
		},
		{
			name: "missing category",
			args: map[string]interface{}{
				"amount": 500.00,
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			args: map[string]interface{}{
				"category": "groceries",
				"amount":   0,
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			args: map[string]interface{}{
				"category": "groceries",
				"amount":   -100.00,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleSetBudget(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_GetBudgets(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name:    "get budgets",
			wantErr: false,
		},
		{
			name: "no budgets set",
			setup: func(m *MockFinanceSpace) {
				m.GetBudgetsFunc = func() map[finance.Category]float64 {
					return map[finance.Category]float64{}
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleGetBudgets(ctx, []byte("{}"))

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_CreateLinkToken(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name: "create link token successfully",
			args: map[string]interface{}{
				"user_id": "user-123",
			},
			wantErr: false,
		},
		{
			name:    "missing user_id",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "create link token failure",
			args: map[string]interface{}{
				"user_id": "user-123",
			},
			setup: func(m *MockFinanceSpace) {
				m.CreateLinkTokenFunc = func(ctx context.Context, userID string) (string, error) {
					return "", errors.New("failed to create link token")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleCreateLinkToken(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_SearchTransactions(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]interface{}
		setup   func(*MockFinanceSpace)
		wantErr bool
	}{
		{
			name: "search transactions successfully",
			args: map[string]interface{}{
				"query": "Netflix",
			},
			wantErr: false,
		},
		{
			name: "search with limit",
			args: map[string]interface{}{
				"query": "grocery",
				"limit": 5,
			},
			wantErr: false,
		},
		{
			name:    "missing query",
			args:    map[string]interface{}{},
			wantErr: true,
		},
		{
			name: "not connected",
			args: map[string]interface{}{
				"query": "test",
			},
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool {
					return false
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			argsJSON, _ := json.Marshal(tt.args)
			result, err := srv.handleSearchTransactions(ctx, argsJSON)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantErr {
				if result == nil || !result.IsError {
					t.Error("expected error result")
				}
				return
			}

			if result.IsError {
				t.Errorf("unexpected error result: %s", result.Content[0].Text)
			}
		})
	}
}

func TestFinanceServer_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	// Test that operations fail gracefully with nil space
	result, err := srv.handleListAccounts(ctx, []byte("{}"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil space")
	}
}

func TestFinanceServer_ToolRegistration(t *testing.T) {
	mock := &MockFinanceSpace{}
	srv := NewWithMockSpace(mock)

	expectedTools := []string{
		"finance.list_accounts",
		"finance.get_balance",
		"finance.list_transactions",
		"finance.spending_summary",
		"finance.recurring",
		"finance.insights",
		"finance.connections",
		"finance.set_budget",
		"finance.get_budgets",
		"finance.create_link_token",
		"finance.search",
	}

	tools := srv.Registry().ListTools()
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolMap[expected] {
			t.Errorf("expected tool %q not registered", expected)
		}
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
}

// Test helper functions
func TestStringToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello", "hello"},
		{"UPPER", "upper"},
		{"MixedCase", "mixedcase"},
		{"123ABC", "123abc"},
	}

	for _, tt := range tests {
		got := stringToLower(tt.input)
		if got != tt.expected {
			t.Errorf("stringToLower(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	// Note: containsIgnoreCase expects substr to already be lowercase
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"Hello World", "world", true},
		{"Hello World", "hello", true},
		{"Hello World", "xyz", false},
		{"Netflix", "net", true},
		{"", "test", false},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.expected {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.expected)
		}
	}
}

// ============================================================================
// Resource Handler Tests
// ============================================================================

func TestFinanceServer_SummaryResource(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
		check   func(t *testing.T, content string)
	}{
		{
			name: "get summary resource when connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool { return true }
			},
			wantErr: false,
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, `"status": "connected"`) {
					t.Error("expected status connected in content")
				}
			},
		},
		{
			name: "get summary resource when not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool { return false }
			},
			wantErr: false,
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, "not_connected") {
					t.Error("expected not_connected in content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleSummaryResource(ctx, "finance://summary")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.URI != "finance://summary" {
				t.Errorf("expected URI 'finance://summary', got %q", result.URI)
			}
			if result.MimeType != "application/json" {
				t.Errorf("expected MimeType 'application/json', got %q", result.MimeType)
			}
			if tt.check != nil {
				tt.check(t, result.Text)
			}
		})
	}
}

func TestFinanceServer_SummaryResource_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	result, err := srv.handleSummaryResource(ctx, "finance://summary")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !strings.Contains(result.Text, "not_connected") {
		t.Error("expected not_connected in content")
	}
}

func TestFinanceServer_MonthlyResource(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*MockFinanceSpace)
		wantErr bool
		check   func(t *testing.T, content string)
	}{
		{
			name: "get monthly resource when connected with data",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool { return true }
			},
			wantErr: false,
			check: func(t *testing.T, content string) {
				if strings.Contains(content, "not_connected") {
					t.Error("unexpected not_connected in content")
				}
			},
		},
		{
			name: "get monthly resource when not connected",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool { return false }
			},
			wantErr: false,
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, "not_connected") {
					t.Error("expected not_connected in content")
				}
			},
		},
		{
			name: "get monthly resource with no spending data",
			setup: func(m *MockFinanceSpace) {
				m.IsConnectedFunc = func() bool { return true }
				m.GetSpendingSummaryFunc = func(period string) *finance.SpendingSummary {
					return nil
				}
			},
			wantErr: false,
			check: func(t *testing.T, content string) {
				if !strings.Contains(content, "no_data") {
					t.Error("expected no_data in content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockFinanceSpace{}
			if tt.setup != nil {
				tt.setup(mock)
			}

			srv := NewWithMockSpace(mock)
			ctx := context.Background()

			result, err := srv.handleMonthlyResource(ctx, "finance://monthly")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.URI != "finance://monthly" {
				t.Errorf("expected URI 'finance://monthly', got %q", result.URI)
			}
			if result.MimeType != "application/json" {
				t.Errorf("expected MimeType 'application/json', got %q", result.MimeType)
			}
			if tt.check != nil {
				tt.check(t, result.Text)
			}
		})
	}
}

func TestFinanceServer_MonthlyResource_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	result, err := srv.handleMonthlyResource(ctx, "finance://monthly")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !strings.Contains(result.Text, "not_connected") {
		t.Error("expected not_connected in content")
	}
}

func TestFinanceServer_ResourceRegistration(t *testing.T) {
	mock := &MockFinanceSpace{}
	srv := NewWithMockSpace(mock)

	resources := srv.Registry().ListResources()

	expectedResources := map[string]string{
		"finance://summary": "Financial Summary",
		"finance://monthly": "Monthly Report",
	}

	for _, r := range resources {
		if expectedName, ok := expectedResources[r.URI]; ok {
			if r.Name != expectedName {
				t.Errorf("resource %q name = %q, want %q", r.URI, r.Name, expectedName)
			}
			delete(expectedResources, r.URI)
		}
	}

	for uri := range expectedResources {
		t.Errorf("expected resource %q not registered", uri)
	}
}

// ============================================================================
// Additional Nil Space Handler Tests
// ============================================================================

func TestFinanceServer_GetConnections_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	result, err := srv.handleGetConnections(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil space")
	}
}

func TestFinanceServer_SetBudget_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"category": "groceries",
		"amount":   500.00,
	})
	result, err := srv.handleSetBudget(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil space")
	}
}

func TestFinanceServer_GetBudgets_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	result, err := srv.handleGetBudgets(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil space")
	}
}

func TestFinanceServer_CreateLinkToken_NilSpace(t *testing.T) {
	srv := NewWithMockSpace(nil)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"user_id": "user-123",
	})
	result, err := srv.handleCreateLinkToken(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil space")
	}
}

// ============================================================================
// Additional Handler Edge Case Tests
// ============================================================================

func TestFinanceServer_ListTransactions_LimitApplied(t *testing.T) {
	// Create more transactions than the limit
	manyTransactions := make([]*finance.CategorizedTransaction, 100)
	for i := 0; i < 100; i++ {
		manyTransactions[i] = &finance.CategorizedTransaction{
			Transaction: finance.Transaction{
				TransactionID: fmt.Sprintf("tx-%03d", i),
				AccountID:     "acc-001",
				Amount:        float64(i * 10),
				Date:          "2024-01-15",
				Name:          fmt.Sprintf("Transaction %d", i),
				MerchantName:  fmt.Sprintf("Merchant %d", i),
			},
			QLCategory: finance.CategoryOther,
		}
	}

	mock := &MockFinanceSpace{
		GetTransactionsFunc: func(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
			return manyTransactions
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"limit": 10,
	})
	result, err := srv.handleListTransactions(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	// Verify the result contains only 10 transactions
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	count, ok := response["count"].(float64)
	if !ok || int(count) != 10 {
		t.Errorf("expected count 10, got %v", response["count"])
	}
}

func TestFinanceServer_ListTransactions_AccountFilter(t *testing.T) {
	mock := &MockFinanceSpace{
		GetTransactionsFunc: func(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
			if filter.AccountID != "acc-001" {
				t.Errorf("expected account_id 'acc-001', got %q", filter.AccountID)
			}
			return sampleTransactions()
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"account_id": "acc-001",
	})
	result, err := srv.handleListTransactions(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}
}

func TestFinanceServer_SearchTransactions_LimitHit(t *testing.T) {
	// Create many matching transactions
	manyTransactions := make([]*finance.CategorizedTransaction, 50)
	for i := 0; i < 50; i++ {
		manyTransactions[i] = &finance.CategorizedTransaction{
			Transaction: finance.Transaction{
				TransactionID: fmt.Sprintf("tx-%03d", i),
				AccountID:     "acc-001",
				Amount:        float64(i * 10),
				Date:          "2024-01-15",
				Name:          fmt.Sprintf("Netflix Subscription %d", i),
				MerchantName:  "Netflix",
			},
			QLCategory: finance.CategorySubscription,
		}
	}

	mock := &MockFinanceSpace{
		GetTransactionsFunc: func(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
			return manyTransactions
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"query": "netflix",
		"limit": 5,
	})
	result, err := srv.handleSearchTransactions(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	// Verify the result contains only 5 matches
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	count, ok := response["count"].(float64)
	if !ok || int(count) != 5 {
		t.Errorf("expected count 5, got %v", response["count"])
	}
}

func TestFinanceServer_SearchTransactions_MerchantMatch(t *testing.T) {
	mock := &MockFinanceSpace{
		GetTransactionsFunc: func(filter finance.TransactionFilter) []*finance.CategorizedTransaction {
			return []*finance.CategorizedTransaction{
				{
					Transaction: finance.Transaction{
						TransactionID: "tx-001",
						Name:          "Some purchase",
						MerchantName:  "Whole Foods Market",
						Amount:        45.00,
						Date:          "2024-01-15",
					},
					QLCategory: finance.CategoryGroceries,
				},
			}
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	argsJSON, _ := json.Marshal(map[string]interface{}{
		"query": "whole foods",
	})
	result, err := srv.handleSearchTransactions(ctx, argsJSON)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	count, ok := response["count"].(float64)
	if !ok || int(count) != 1 {
		t.Errorf("expected count 1, got %v", response["count"])
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestIndexOfString(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "xyz", -1},
		{"abcabc", "abc", 0},
		{"", "test", -1},
		{"test", "", 0},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s-%s", tt.s, tt.substr), func(t *testing.T) {
			got := indexOfString(tt.s, tt.substr)
			if got != tt.expected {
				t.Errorf("indexOfString(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.expected)
			}
		})
	}
}

func TestStringToLower_Extended(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"Hello World!", "hello world!"},
		{"ABC123xyz", "abc123xyz"},
		{"ALL CAPS", "all caps"},
		{"already lowercase", "already lowercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stringToLower(tt.input)
			if got != tt.expected {
				t.Errorf("stringToLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// ============================================================================
// Benchmark Tests
// ============================================================================

func BenchmarkStringToLower(b *testing.B) {
	inputs := []string{
		"Hello World",
		"COMPLETELY UPPERCASE STRING",
		"Mixed Case String With Numbers 123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			stringToLower(input)
		}
	}
}

func BenchmarkContainsIgnoreCase(b *testing.B) {
	pairs := []struct {
		s      string
		substr string
	}{
		{"Hello World", "world"},
		{"Netflix Subscription Payment", "netflix"},
		{"Whole Foods Market Purchase", "foods"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range pairs {
			containsIgnoreCase(p.s, p.substr)
		}
	}
}

func BenchmarkIndexOfString(b *testing.B) {
	s := "This is a long string with some content to search through"
	substrs := []string{"long", "content", "search", "xyz"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, substr := range substrs {
			indexOfString(s, substr)
		}
	}
}

func BenchmarkHandleListAccounts(b *testing.B) {
	mock := &MockFinanceSpace{}
	srv := NewWithMockSpace(mock)
	ctx := context.Background()
	argsJSON := []byte("{}")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.handleListAccounts(ctx, argsJSON)
	}
}

func BenchmarkHandleSearchTransactions(b *testing.B) {
	mock := &MockFinanceSpace{}
	srv := NewWithMockSpace(mock)
	ctx := context.Background()
	argsJSON, _ := json.Marshal(map[string]interface{}{
		"query": "netflix",
		"limit": 10,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		srv.handleSearchTransactions(ctx, argsJSON)
	}
}

// ============================================================================
// Additional Coverage Tests
// ============================================================================

func TestFinanceServer_GetBalance_WithData(t *testing.T) {
	mock := &MockFinanceSpace{
		GetNetWorthFunc: func() (assets, liabilities, netWorth float64) {
			return 50000.00, 10000.00, 40000.00
		},
		GetTotalBalanceFunc: func() float64 {
			return 45000.00
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	result, err := srv.handleGetBalance(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["net_worth"] != 40000.00 {
		t.Errorf("expected net_worth 40000, got %v", response["net_worth"])
	}
}

func TestFinanceServer_GetRecurring_WithData(t *testing.T) {
	mock := &MockFinanceSpace{
		GetRecurringTransactionsFunc: func() []*finance.RecurringTransaction {
			return []*finance.RecurringTransaction{
				{
					ID:           "rec-001",
					MerchantName: "Netflix",
					Category:     finance.CategorySubscription,
					Amount:       15.99,
					Frequency:    "monthly",
					NextExpected: time.Now().AddDate(0, 1, 0),
					LastSeen:     time.Now(),
					IsActive:     true,
					Transactions: []string{"tx-001", "tx-002", "tx-003"},
				},
			}
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	result, err := srv.handleGetRecurring(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"] != float64(1) {
		t.Errorf("expected count 1, got %v", response["count"])
	}
}

func TestFinanceServer_GetInsights_WithData(t *testing.T) {
	mock := &MockFinanceSpace{
		GetInsightsFunc: func() []*finance.Insight {
			return []*finance.Insight{
				{
					ID:          "insight-001",
					Type:        finance.InsightTypeBudgetAlert,
					Title:       "Budget Alert",
					Description: "You've exceeded your dining budget",
					Severity:    finance.SeverityAlert,
					Amount:      50.00,
					Category:    finance.CategoryDining,
				},
				{
					ID:          "insight-002",
					Type:        finance.InsightTypeAnomaly,
					Title:       "Unusual Spending",
					Description: "Higher than usual grocery spending",
					Severity:    finance.SeverityWarning,
					Amount:      100.00,
					Category:    finance.CategoryGroceries,
				},
			}
		},
	}

	srv := NewWithMockSpace(mock)
	ctx := context.Background()

	result, err := srv.handleGetInsights(ctx, []byte("{}"))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %s", result.Content[0].Text)
	}

	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if response["count"] != float64(2) {
		t.Errorf("expected count 2, got %v", response["count"])
	}
}
