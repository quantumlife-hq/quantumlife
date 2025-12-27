// Package finance implements the finance space integration.
package finance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/spaces"
)

// Space implements the Finance data source
type Space struct {
	id           core.SpaceID
	name         string
	defaultHatID core.HatID

	// Plaid
	plaidClient *PlaidClient
	connections []*Connection

	// Processing
	categorizer       *Categorizer
	recurringDetector *RecurringDetector
	insightsEngine    *InsightsEngine

	// Cached data
	accounts     []Account
	transactions []*CategorizedTransaction
	recurring    []*RecurringTransaction
	insights     []*Insight

	// State
	connected  bool
	syncStatus spaces.SyncStatus
	syncCursor string

	mu sync.RWMutex
}

// SpaceConfig for creating a Finance space
type SpaceConfig struct {
	ID           core.SpaceID
	Name         string
	DefaultHatID core.HatID
	PlaidConfig  PlaidConfig
	LLMClient    *llm.OllamaClient
	Budgets      map[Category]float64
}

// NewSpace creates a new Finance space
func NewSpace(cfg SpaceConfig) *Space {
	categorizer := NewCategorizer(CategorizerConfig{
		LLMClient: cfg.LLMClient,
		UseAI:     cfg.LLMClient != nil,
	})

	insightsEngine := NewInsightsEngine(InsightsConfig{
		LLMClient: cfg.LLMClient,
		Budgets:   cfg.Budgets,
	})

	return &Space{
		id:                cfg.ID,
		name:              cfg.Name,
		defaultHatID:      cfg.DefaultHatID,
		plaidClient:       NewPlaidClient(cfg.PlaidConfig),
		categorizer:       categorizer,
		recurringDetector: NewRecurringDetector(),
		insightsEngine:    insightsEngine,
		connections:       make([]*Connection, 0),
		syncStatus: spaces.SyncStatus{
			Status: "idle",
		},
	}
}

// ID returns the space ID
func (s *Space) ID() core.SpaceID {
	return s.id
}

// Type returns the space type
func (s *Space) Type() core.SpaceType {
	return core.SpaceTypeFinance
}

// Provider returns the provider name
func (s *Space) Provider() string {
	return "plaid"
}

// Name returns the space name
func (s *Space) Name() string {
	return s.name
}

// IsConnected returns connection status
func (s *Space) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected && len(s.connections) > 0
}

// CreateLinkToken creates a Plaid Link token for connecting accounts
func (s *Space) CreateLinkToken(ctx context.Context, userID string) (string, error) {
	resp, err := s.plaidClient.CreateLinkToken(ctx, userID)
	if err != nil {
		return "", err
	}
	return resp.LinkToken, nil
}

// LinkAccount completes the account linking process
func (s *Space) LinkAccount(ctx context.Context, publicToken, institutionID string) (*Connection, error) {
	// Exchange public token for access token
	exchangeResp, err := s.plaidClient.ExchangePublicToken(ctx, publicToken)
	if err != nil {
		return nil, fmt.Errorf("exchange token: %w", err)
	}

	// Get institution name
	institution, err := s.plaidClient.GetInstitution(ctx, institutionID)
	institutionName := institutionID
	if err == nil {
		institutionName = institution.Name
	}

	// Get accounts
	accountsResp, err := s.plaidClient.GetAccounts(ctx, exchangeResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("get accounts: %w", err)
	}

	now := time.Now()
	connection := &Connection{
		ID:              fmt.Sprintf("conn_%d", now.UnixNano()),
		ItemID:          exchangeResp.ItemID,
		InstitutionID:   institutionID,
		InstitutionName: institutionName,
		AccessToken:     exchangeResp.AccessToken,
		Status:          ConnectionStatusActive,
		Accounts:        accountsResp.Accounts,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.mu.Lock()
	s.connections = append(s.connections, connection)
	s.accounts = append(s.accounts, accountsResp.Accounts...)
	s.connected = true
	s.mu.Unlock()

	return connection, nil
}

// Connect reconnects with existing credentials
func (s *Space) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.connections) == 0 {
		return fmt.Errorf("no connections configured - use LinkAccount first")
	}

	// Verify connections are still valid
	for _, conn := range s.connections {
		if conn.Status != ConnectionStatusActive {
			continue
		}

		_, err := s.plaidClient.GetAccounts(ctx, conn.AccessToken)
		if err != nil {
			conn.Status = ConnectionStatusError
			continue
		}
	}

	s.connected = true
	s.syncStatus.Status = "idle"
	return nil
}

// Disconnect removes all connections
func (s *Space) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		s.plaidClient.RemoveItem(ctx, conn.AccessToken)
	}

	s.connections = nil
	s.accounts = nil
	s.transactions = nil
	s.recurring = nil
	s.connected = false
	s.syncStatus.Status = "disconnected"

	return nil
}

// Sync fetches latest transactions
func (s *Space) Sync(ctx context.Context) (*spaces.SyncResult, error) {
	s.mu.Lock()
	if !s.connected {
		s.mu.Unlock()
		return nil, fmt.Errorf("not connected")
	}
	s.syncStatus.Status = "syncing"
	connections := s.connections
	s.mu.Unlock()

	start := time.Now()
	result := &spaces.SyncResult{}

	// Sync each connection
	var allTransactions []Transaction
	for _, conn := range connections {
		if conn.Status != ConnectionStatusActive {
			continue
		}

		// Use sync API for incremental updates
		syncResp, err := s.plaidClient.SyncTransactions(ctx, conn.AccessToken, conn.SyncCursor)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("sync %s: %w", conn.InstitutionName, err))
			continue
		}

		allTransactions = append(allTransactions, syncResp.Added...)
		allTransactions = append(allTransactions, syncResp.Modified...)
		conn.SyncCursor = syncResp.NextCursor
		conn.LastSync = time.Now()

		// Update accounts
		accountsResp, err := s.plaidClient.Balance(ctx, conn.AccessToken)
		if err == nil {
			conn.Accounts = accountsResp.Accounts
		}

		result.NewItems += len(syncResp.Added)
		result.UpdatedItems += len(syncResp.Modified)
	}

	// Categorize transactions
	categorized := s.categorizer.BatchCategorize(allTransactions)

	// Detect recurring transactions
	recurring := s.recurringDetector.DetectRecurring(categorized)

	// Generate insights
	insights := s.generateInsights(categorized, recurring)

	// Update cached data
	s.mu.Lock()
	s.transactions = append(s.transactions, categorized...)
	s.recurring = recurring
	s.insights = insights
	s.syncStatus.Status = "idle"
	s.syncStatus.LastSync = time.Now()
	s.syncStatus.ItemCount = len(s.transactions)
	s.mu.Unlock()

	result.Duration = time.Since(start)
	result.Cursor = time.Now().Format(time.RFC3339)

	return result, nil
}

// generateInsights creates financial insights
func (s *Space) generateInsights(transactions []*CategorizedTransaction, recurring []*RecurringTransaction) []*Insight {
	var insights []*Insight

	// Category insights
	insights = append(insights, s.insightsEngine.GenerateCategoryInsights(transactions)...)

	// Anomalies
	insights = append(insights, s.insightsEngine.DetectAnomalies(transactions)...)

	// Bill reminders
	insights = append(insights, s.insightsEngine.GenerateBillReminders(recurring)...)

	// Savings opportunities
	insights = append(insights, s.insightsEngine.FindSavingsOpportunities(transactions, recurring)...)

	return insights
}

// GetSyncStatus returns the current sync status
func (s *Space) GetSyncStatus() spaces.SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncStatus
}

// GetAccounts returns all connected accounts
func (s *Space) GetAccounts() []Account {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accounts
}

// GetTransactions returns transactions with optional filtering
func (s *Space) GetTransactions(filter TransactionFilter) []*CategorizedTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*CategorizedTransaction
	for _, tx := range s.transactions {
		if filter.Matches(tx) {
			result = append(result, tx)
		}
	}
	return result
}

// TransactionFilter for querying transactions
type TransactionFilter struct {
	Categories []Category
	StartDate  time.Time
	EndDate    time.Time
	MinAmount  float64
	MaxAmount  float64
	AccountID  string
	Recurring  *bool
}

// Matches checks if a transaction matches the filter
func (f TransactionFilter) Matches(tx *CategorizedTransaction) bool {
	// Category filter
	if len(f.Categories) > 0 {
		found := false
		for _, cat := range f.Categories {
			if tx.QLCategory == cat {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Date filter
	txDate, _ := time.Parse("2006-01-02", tx.Date)
	if !f.StartDate.IsZero() && txDate.Before(f.StartDate) {
		return false
	}
	if !f.EndDate.IsZero() && txDate.After(f.EndDate) {
		return false
	}

	// Amount filter
	if f.MinAmount > 0 && tx.Amount < f.MinAmount {
		return false
	}
	if f.MaxAmount > 0 && tx.Amount > f.MaxAmount {
		return false
	}

	// Account filter
	if f.AccountID != "" && tx.AccountID != f.AccountID {
		return false
	}

	// Recurring filter
	if f.Recurring != nil && tx.IsRecurring != *f.Recurring {
		return false
	}

	return true
}

// GetRecurringTransactions returns detected recurring transactions
func (s *Space) GetRecurringTransactions() []*RecurringTransaction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.recurring
}

// GetInsights returns financial insights
func (s *Space) GetInsights() []*Insight {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.insights
}

// GetSpendingSummary returns a spending summary for the period
func (s *Space) GetSpendingSummary(period string) *SpendingSummary {
	s.mu.RLock()
	transactions := s.transactions
	s.mu.RUnlock()

	return s.insightsEngine.GenerateSpendingSummary(transactions, period)
}

// SetBudget sets a budget for a category
func (s *Space) SetBudget(category Category, amount float64) {
	s.insightsEngine.SetBudget(category, amount)
}

// GetBudgets returns all configured budgets
func (s *Space) GetBudgets() map[Category]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	budgets := make(map[Category]float64)
	for _, cat := range AllCategories() {
		if amount, ok := s.insightsEngine.GetBudget(cat); ok {
			budgets[cat] = amount
		}
	}
	return budgets
}

// GetConnections returns all bank connections
func (s *Space) GetConnections() []*Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connections
}

// RemoveConnection removes a bank connection
func (s *Space) RemoveConnection(ctx context.Context, connectionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, conn := range s.connections {
		if conn.ID == connectionID {
			s.plaidClient.RemoveItem(ctx, conn.AccessToken)
			s.connections = append(s.connections[:i], s.connections[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("connection not found: %s", connectionID)
}

// RefreshConnection updates account data for a connection
func (s *Space) RefreshConnection(ctx context.Context, connectionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, conn := range s.connections {
		if conn.ID != connectionID {
			continue
		}

		// Refresh accounts
		accountsResp, err := s.plaidClient.Balance(ctx, conn.AccessToken)
		if err != nil {
			conn.Status = ConnectionStatusError
			return fmt.Errorf("refresh accounts: %w", err)
		}

		conn.Accounts = accountsResp.Accounts
		conn.UpdatedAt = time.Now()
		conn.Status = ConnectionStatusActive
		return nil
	}

	return fmt.Errorf("connection not found: %s", connectionID)
}

// GetTotalBalance calculates total balance across all accounts
func (s *Space) GetTotalBalance() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var total float64
	for _, account := range s.accounts {
		total += account.Balances.Current
	}
	return total
}

// GetNetWorth calculates net worth (assets - liabilities)
func (s *Space) GetNetWorth() (assets, liabilities, netWorth float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, account := range s.accounts {
		switch account.Type {
		case "depository", "investment":
			assets += account.Balances.Current
		case "credit", "loan":
			liabilities += account.Balances.Current
		}
	}

	netWorth = assets - liabilities
	return
}

// TransactionToItem converts a transaction to a core.Item
func TransactionToItem(tx *CategorizedTransaction, spaceID core.SpaceID, hatID core.HatID) *core.Item {
	txDate, _ := time.Parse("2006-01-02", tx.Date)

	return &core.Item{
		ID:         core.ItemID(fmt.Sprintf("fin_%s", tx.TransactionID)),
		SpaceID:    spaceID,
		Type:       core.ItemTypeTransaction,
		ExternalID: tx.TransactionID,
		Subject:    tx.Name,
		Body:       fmt.Sprintf("$%.2f at %s", tx.Amount, tx.MerchantName),
		From:       tx.MerchantName,
		Timestamp:  txDate,
		HatID:      hatID,
		Status:     core.ItemStatusActioned,
	}
}
