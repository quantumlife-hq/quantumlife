// Package test contains Week 4 integration tests.
package test

import (
	"context"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/finance"
	"github.com/quantumlife/quantumlife/internal/mesh"
)

// ==================== MESH TESTS ====================

// TestAgentKeyPairGeneration tests Ed25519 key pair generation
func TestAgentKeyPairGeneration(t *testing.T) {
	keyPair, err := mesh.GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if len(keyPair.PublicKey) != 32 {
		t.Errorf("Expected 32-byte public key, got %d", len(keyPair.PublicKey))
	}

	if len(keyPair.PrivateKey) != 64 {
		t.Errorf("Expected 64-byte private key, got %d", len(keyPair.PrivateKey))
	}

	t.Log("Ed25519 key pair generated successfully")
}

// TestAgentCardCreation tests creating and signing agent cards
func TestAgentCardCreation(t *testing.T) {
	keyPair, err := mesh.GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	capabilities := []mesh.AgentCapability{
		mesh.CapabilityCalendar,
		mesh.CapabilityEmail,
		mesh.CapabilityFinance,
	}

	card := mesh.NewAgentCard("agent-001", "Alice's Agent", "http://localhost:8090", keyPair, capabilities)

	if card.ID != "agent-001" {
		t.Errorf("Expected ID 'agent-001', got '%s'", card.ID)
	}

	if card.Name != "Alice's Agent" {
		t.Errorf("Expected name 'Alice's Agent', got '%s'", card.Name)
	}

	if !card.HasCapability(mesh.CapabilityCalendar) {
		t.Error("Expected agent to have calendar capability")
	}

	if card.HasCapability(mesh.CapabilityTasks) {
		t.Error("Agent should not have tasks capability")
	}

	t.Logf("Agent card created: %s (%s)", card.Name, card.Fingerprint())
}

// TestAgentCardSignature tests card signing and verification
func TestAgentCardSignature(t *testing.T) {
	keyPair, err := mesh.GenerateAgentKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	card := mesh.NewAgentCard("test-agent", "Test Agent", "http://localhost:8090", keyPair, nil)

	// Sign the card
	if err := card.Sign(keyPair.PrivateKey); err != nil {
		t.Fatalf("Failed to sign card: %v", err)
	}

	if len(card.Signature) == 0 {
		t.Error("Expected non-empty signature")
	}

	// Verify signature
	if !card.Verify() {
		t.Error("Card signature verification failed")
	}

	t.Log("Agent card signature verified successfully")
}

// TestAgentRelationships tests relationship management
func TestAgentRelationships(t *testing.T) {
	keyPair, _ := mesh.GenerateAgentKeyPair()
	card := mesh.NewAgentCard("agent-1", "Agent 1", "http://localhost:8090", keyPair, nil)

	// Add a relationship
	rel := mesh.Relationship{
		AgentID:   "agent-2",
		AgentName: "Agent 2",
		Type:      mesh.RelationshipSpouse,
		Permissions: []mesh.Permission{
			{Capability: mesh.CapabilityCalendar, Level: mesh.PermissionFull},
			{Capability: mesh.CapabilityFinance, Level: mesh.PermissionView},
		},
		Verified: true,
		Since:    time.Now(),
	}
	card.AddRelationship(rel)

	// Check relationship exists
	found := card.GetRelationship("agent-2")
	if found == nil {
		t.Fatal("Expected to find relationship")
	}

	if found.Type != mesh.RelationshipSpouse {
		t.Errorf("Expected spouse relationship, got %s", found.Type)
	}

	// Check permissions
	calLevel := card.GetPermissionLevel("agent-2", mesh.CapabilityCalendar)
	if calLevel != mesh.PermissionFull {
		t.Errorf("Expected full calendar permission, got %s", calLevel)
	}

	finLevel := card.GetPermissionLevel("agent-2", mesh.CapabilityFinance)
	if finLevel != mesh.PermissionView {
		t.Errorf("Expected view finance permission, got %s", finLevel)
	}

	// Check access
	if !card.CanAccess("agent-2", mesh.CapabilityCalendar, mesh.PermissionModify) {
		t.Error("Expected access to calendar with modify level")
	}

	if card.CanAccess("agent-2", mesh.CapabilityFinance, mesh.PermissionModify) {
		t.Error("Should not have modify access to finance")
	}

	t.Log("Relationship management working correctly")
}

// TestPairingRequest tests creating pairing requests
func TestPairingRequest(t *testing.T) {
	keyPair, _ := mesh.GenerateAgentKeyPair()
	card := mesh.NewAgentCard("requester", "Requester", "http://localhost:8090", keyPair, nil)
	card.Sign(keyPair.PrivateKey)

	req, err := mesh.NewPairingRequest(card, mesh.RelationshipSpouse, "Let's pair!", keyPair.PrivateKey)
	if err != nil {
		t.Fatalf("Failed to create pairing request: %v", err)
	}

	if req.Relationship != mesh.RelationshipSpouse {
		t.Errorf("Expected spouse relationship, got %s", req.Relationship)
	}

	if !req.Verify() {
		t.Error("Pairing request verification failed")
	}

	t.Log("Pairing request created and verified")
}

// TestX25519KeyExchange tests key exchange
func TestX25519KeyExchange(t *testing.T) {
	// Generate two key pairs
	keyPair1, err := mesh.GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 1: %v", err)
	}

	keyPair2, err := mesh.GenerateX25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair 2: %v", err)
	}

	// Keys should be different
	if keyPair1.PublicKey == keyPair2.PublicKey {
		t.Error("Generated identical public keys")
	}

	t.Log("X25519 key pairs generated successfully")
}

// TestChannelCreation tests channel creation
func TestChannelCreation(t *testing.T) {
	channel, err := mesh.NewChannel(mesh.ChannelConfig{
		LocalAgentID:  "agent-1",
		RemoteAgentID: "agent-2",
	})
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	if channel.LocalAgent != "agent-1" {
		t.Errorf("Expected local agent 'agent-1', got '%s'", channel.LocalAgent)
	}

	if channel.RemoteAgent != "agent-2" {
		t.Errorf("Expected remote agent 'agent-2', got '%s'", channel.RemoteAgent)
	}

	if channel.State != mesh.ChannelStateNew {
		t.Errorf("Expected state 'new', got '%s'", channel.State)
	}

	t.Logf("Channel created: %s", channel.ID)
}

// TestChannelEncryption tests end-to-end encryption
func TestChannelEncryption(t *testing.T) {
	// Create two channels (simulating two agents)
	channel1, _ := mesh.NewChannel(mesh.ChannelConfig{
		LocalAgentID:  "agent-1",
		RemoteAgentID: "agent-2",
	})

	channel2, _ := mesh.NewChannel(mesh.ChannelConfig{
		LocalAgentID:  "agent-2",
		RemoteAgentID: "agent-1",
	})

	// Exchange public keys
	pub1 := channel1.GetLocalPublicKey()
	pub2 := channel2.GetLocalPublicKey()

	if err := channel1.SetRemotePublicKey(pub2); err != nil {
		t.Fatalf("Failed to set remote key on channel 1: %v", err)
	}

	if err := channel2.SetRemotePublicKey(pub1); err != nil {
		t.Fatalf("Failed to set remote key on channel 2: %v", err)
	}

	// Both channels should be established
	if !channel1.IsEstablished() || !channel2.IsEstablished() {
		t.Error("Channels should be established after key exchange")
	}

	// Test encryption/decryption
	plaintext := []byte("Hello, secure world!")

	ciphertext, nonce, err := channel1.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := channel2.Decrypt(ciphertext, nonce)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decryption mismatch: got '%s', expected '%s'", decrypted, plaintext)
	}

	t.Log("End-to-end encryption working correctly")
}

// TestChannelManager tests channel management
func TestChannelManager(t *testing.T) {
	manager := mesh.NewChannelManager()

	// Create/get channel
	ch1, err := manager.GetOrCreateChannel("agent-1", "agent-2")
	if err != nil {
		t.Fatalf("Failed to create channel: %v", err)
	}

	// Get same channel again
	ch2, _ := manager.GetOrCreateChannel("agent-1", "agent-2")
	if ch1.ID != ch2.ID {
		t.Error("Expected same channel on second call")
	}

	// List channels
	channels := manager.ListChannels()
	if len(channels) != 1 {
		t.Errorf("Expected 1 channel, got %d", len(channels))
	}

	// Remove channel
	manager.RemoveChannel(ch1.ID)
	channels = manager.ListChannels()
	if len(channels) != 0 {
		t.Error("Expected 0 channels after removal")
	}

	t.Log("Channel manager working correctly")
}

// TestNegotiationEngine tests negotiation protocol
func TestNegotiationEngine(t *testing.T) {
	cfg := mesh.DefaultNegotiationConfig()
	cfg.AgentID = "agent-1"
	engine := mesh.NewNegotiationEngine(cfg)

	ctx := context.Background()

	// Create a proposal
	proposal := mesh.ScheduleProposal{
		EventType:    "meeting",
		Title:        "Family Dinner",
		StartTime:    time.Now().Add(24 * time.Hour),
		EndTime:      time.Now().Add(26 * time.Hour),
		Participants: []string{"agent-1", "agent-2"},
		Flexible:     true,
	}

	neg, err := engine.Propose(ctx, mesh.NegotiationSchedule, "agent-2", proposal, mesh.PriorityNormal)
	if err != nil {
		t.Fatalf("Failed to create proposal: %v", err)
	}

	if neg.Status != mesh.NegotiationStatusPending {
		t.Errorf("Expected pending status, got %s", neg.Status)
	}

	if neg.Initiator != "agent-1" {
		t.Errorf("Expected initiator 'agent-1', got '%s'", neg.Initiator)
	}

	// Accept the negotiation
	if err := engine.Respond(ctx, neg.ID, true, nil); err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}

	updated, _ := engine.GetNegotiation(neg.ID)
	if updated.Status != mesh.NegotiationStatusAccepted {
		t.Errorf("Expected accepted status, got %s", updated.Status)
	}

	t.Logf("Negotiation %s completed", neg.ID)
}

// TestScheduleNegotiator tests schedule-specific negotiation
func TestScheduleNegotiator(t *testing.T) {
	engine := mesh.NewNegotiationEngine(mesh.DefaultNegotiationConfig())
	negotiator := mesh.NewScheduleNegotiator(engine)

	// Set local availability
	now := time.Now()
	localSlots := []mesh.TimeSlot{
		{Start: now.Add(2 * time.Hour), End: now.Add(4 * time.Hour), Priority: mesh.PriorityHigh},
		{Start: now.Add(8 * time.Hour), End: now.Add(10 * time.Hour), Priority: mesh.PriorityNormal},
	}
	negotiator.SetAvailability(localSlots)

	// Remote availability
	remoteSlots := []mesh.TimeSlot{
		{Start: now.Add(3 * time.Hour), End: now.Add(5 * time.Hour), Priority: mesh.PriorityNormal},
		{Start: now.Add(9 * time.Hour), End: now.Add(11 * time.Hour), Priority: mesh.PriorityHigh},
	}

	// Find common times for a 1-hour meeting
	common := negotiator.FindCommonTime(remoteSlots, time.Hour)

	if len(common) < 2 {
		t.Errorf("Expected at least 2 common slots, got %d", len(common))
	}

	t.Logf("Found %d common time slots", len(common))
}

// TestMeshHub tests hub creation
func TestMeshHub(t *testing.T) {
	keyPair, _ := mesh.GenerateAgentKeyPair()
	card := mesh.NewAgentCard("test-hub", "Test Hub", "http://localhost:8090", keyPair, nil)
	card.Sign(keyPair.PrivateKey)

	hub := mesh.NewHub(mesh.HubConfig{
		AgentCard: card,
		KeyPair:   keyPair,
	})

	if hub == nil {
		t.Fatal("Failed to create hub")
	}

	if hub.AgentCard().ID != "test-hub" {
		t.Errorf("Expected agent ID 'test-hub', got '%s'", hub.AgentCard().ID)
	}

	peers := hub.ListPeers()
	if len(peers) != 0 {
		t.Errorf("Expected 0 peers initially, got %d", len(peers))
	}

	t.Log("Mesh hub created successfully")
}

// ==================== FINANCE TESTS ====================

// TestPlaidConfig tests Plaid configuration
func TestPlaidConfig(t *testing.T) {
	cfg := finance.DefaultPlaidConfig()

	if cfg.Environment != finance.EnvironmentSandbox {
		t.Errorf("Expected sandbox environment, got %s", cfg.Environment)
	}

	if cfg.ClientName != "QuantumLife" {
		t.Errorf("Expected client name 'QuantumLife', got '%s'", cfg.ClientName)
	}

	if len(cfg.CountryCodes) == 0 {
		t.Error("Expected at least one country code")
	}

	if len(cfg.Products) == 0 {
		t.Error("Expected at least one product")
	}

	t.Logf("Plaid config: %s environment, products: %v", cfg.Environment, cfg.Products)
}

// TestPlaidClientCreation tests client creation
func TestPlaidClientCreation(t *testing.T) {
	client := finance.NewPlaidClient(finance.PlaidConfig{
		ClientID:    "test-client-id",
		Secret:      "test-secret",
		Environment: finance.EnvironmentSandbox,
		ClientName:  "QuantumLife Test",
	})

	if client == nil {
		t.Fatal("Failed to create Plaid client")
	}

	t.Log("Plaid client created successfully")
}

// TestCategorizer tests transaction categorization
func TestCategorizer(t *testing.T) {
	categorizer := finance.NewCategorizer(finance.CategorizerConfig{})

	tests := []struct {
		name     string
		merchant string
		expected finance.Category
	}{
		{"Walmart purchase", "WALMART SUPERCENTER", finance.CategoryGroceries},
		{"Starbucks coffee", "STARBUCKS #12345", finance.CategoryDining},
		{"Uber ride", "UBER TRIP", finance.CategoryTransport},
		{"Netflix subscription", "NETFLIX.COM", finance.CategorySubscription},
		{"Shell gas", "SHELL OIL 12345", finance.CategoryTransport},
		{"Amazon shopping", "AMAZON.COM*ABC123", finance.CategoryShopping},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := finance.Transaction{
				TransactionID: "test-123",
				Name:         tt.name,
				MerchantName: tt.merchant,
				Amount:       25.00,
				Date:         "2024-01-15",
			}

			result := categorizer.Categorize(tx)

			if result.QLCategory != tt.expected {
				t.Errorf("Expected category %s, got %s (confidence: %.2f)", tt.expected, result.QLCategory, result.Confidence)
			} else {
				t.Logf("✓ %s -> %s (confidence: %.2f)", tt.merchant, result.QLCategory, result.Confidence)
			}
		})
	}
}

// TestCategorizerPlaidMapping tests Plaid category mapping
func TestCategorizerPlaidMapping(t *testing.T) {
	categorizer := finance.NewCategorizer(finance.CategorizerConfig{})

	tx := finance.Transaction{
		TransactionID: "test-456",
		Name:         "Some Restaurant",
		MerchantName: "OBSCURE RESTAURANT",
		Amount:       45.00,
		Date:         "2024-01-15",
		Category:     []string{"Food and Drink", "Restaurants"},
		PersonalFinanceCategory: finance.PersonalFinanceCategory{
			Primary:  "FOOD_AND_DRINK",
			Detailed: "FOOD_AND_DRINK_RESTAURANTS",
		},
	}

	result := categorizer.Categorize(tx)

	if result.QLCategory != finance.CategoryDining {
		t.Errorf("Expected dining category from Plaid mapping, got %s", result.QLCategory)
	}

	t.Logf("Plaid category mapping: %v -> %s", tx.Category, result.QLCategory)
}

// TestRecurringDetection tests recurring transaction detection
func TestRecurringDetection(t *testing.T) {
	detector := finance.NewRecurringDetector()

	// Create mock recurring transactions
	transactions := []*finance.CategorizedTransaction{
		{Transaction: finance.Transaction{TransactionID: "1", MerchantName: "Netflix", Amount: 15.99, Date: "2024-01-15"}, QLCategory: finance.CategorySubscription},
		{Transaction: finance.Transaction{TransactionID: "2", MerchantName: "Netflix", Amount: 15.99, Date: "2024-02-15"}, QLCategory: finance.CategorySubscription},
		{Transaction: finance.Transaction{TransactionID: "3", MerchantName: "Netflix", Amount: 15.99, Date: "2024-03-15"}, QLCategory: finance.CategorySubscription},
		{Transaction: finance.Transaction{TransactionID: "4", MerchantName: "Spotify", Amount: 9.99, Date: "2024-01-01"}, QLCategory: finance.CategorySubscription},
		{Transaction: finance.Transaction{TransactionID: "5", MerchantName: "Spotify", Amount: 9.99, Date: "2024-02-01"}, QLCategory: finance.CategorySubscription},
		{Transaction: finance.Transaction{TransactionID: "6", MerchantName: "Random Store", Amount: 50.00, Date: "2024-01-20"}, QLCategory: finance.CategoryShopping},
	}

	recurring := detector.DetectRecurring(transactions)

	if len(recurring) < 2 {
		t.Errorf("Expected at least 2 recurring transactions, got %d", len(recurring))
	}

	for _, rec := range recurring {
		t.Logf("Recurring: %s $%.2f (%s)", rec.MerchantName, rec.Amount, rec.Frequency)
	}
}

// TestSpendingSummary tests spending analysis
func TestSpendingSummary(t *testing.T) {
	engine := finance.NewInsightsEngine(finance.InsightsConfig{})

	transactions := []*finance.CategorizedTransaction{
		{Transaction: finance.Transaction{Amount: 100.00}, QLCategory: finance.CategoryGroceries},
		{Transaction: finance.Transaction{Amount: 50.00}, QLCategory: finance.CategoryDining},
		{Transaction: finance.Transaction{Amount: 30.00}, QLCategory: finance.CategoryTransport},
		{Transaction: finance.Transaction{Amount: -2000.00}, QLCategory: finance.CategoryIncome}, // Income
	}

	summary := engine.GenerateSpendingSummary(transactions, "month")

	if summary.TotalSpent != 180.00 {
		t.Errorf("Expected total spent $180.00, got $%.2f", summary.TotalSpent)
	}

	if summary.TotalIncome != 2000.00 {
		t.Errorf("Expected total income $2000.00, got $%.2f", summary.TotalIncome)
	}

	if summary.NetCashFlow != 1820.00 {
		t.Errorf("Expected net cash flow $1820.00, got $%.2f", summary.NetCashFlow)
	}

	t.Logf("Summary: Spent $%.2f, Income $%.2f, Net $%.2f", summary.TotalSpent, summary.TotalIncome, summary.NetCashFlow)
}

// TestBudgetAlerts tests budget threshold alerts
func TestBudgetAlerts(t *testing.T) {
	engine := finance.NewInsightsEngine(finance.InsightsConfig{
		Budgets: map[finance.Category]float64{
			finance.CategoryDining: 200.00,
		},
	})

	// Transactions that exceed budget
	transactions := []*finance.CategorizedTransaction{
		{Transaction: finance.Transaction{Amount: 150.00}, QLCategory: finance.CategoryDining},
		{Transaction: finance.Transaction{Amount: 80.00}, QLCategory: finance.CategoryDining},
	}

	insights := engine.GenerateCategoryInsights(transactions)

	found := false
	for _, insight := range insights {
		if insight.Type == finance.InsightTypeBudgetAlert && insight.Category == finance.CategoryDining {
			found = true
			t.Logf("Budget alert: %s - %s", insight.Title, insight.Description)
		}
	}

	if !found {
		t.Error("Expected budget exceeded alert for dining")
	}
}

// TestFinanceSpace tests finance space creation
func TestFinanceSpace(t *testing.T) {
	space := finance.NewSpace(finance.SpaceConfig{
		ID:           "finance-1",
		Name:         "Personal Finance",
		DefaultHatID: core.HatPersonal,
		PlaidConfig:  finance.DefaultPlaidConfig(),
	})

	if space == nil {
		t.Fatal("Failed to create finance space")
	}

	if space.ID() != "finance-1" {
		t.Errorf("Expected ID 'finance-1', got '%s'", space.ID())
	}

	if space.Type() != core.SpaceTypeFinance {
		t.Errorf("Expected type 'finance', got '%s'", space.Type())
	}

	if space.Provider() != "plaid" {
		t.Errorf("Expected provider 'plaid', got '%s'", space.Provider())
	}

	if space.IsConnected() {
		t.Error("Space should not be connected initially")
	}

	t.Logf("Finance space created: %s (%s)", space.Name(), space.Type())
}

// TestTransactionFilter tests transaction filtering
func TestTransactionFilter(t *testing.T) {
	tx := &finance.CategorizedTransaction{
		Transaction: finance.Transaction{
			TransactionID: "test-1",
			AccountID:     "acc-1",
			Amount:        100.00,
			Date:          "2024-01-15",
		},
		QLCategory:  finance.CategoryDining,
		IsRecurring: false,
	}

	// Test category filter
	filter := finance.TransactionFilter{
		Categories: []finance.Category{finance.CategoryDining},
	}
	if !filter.Matches(tx) {
		t.Error("Should match dining category")
	}

	filter.Categories = []finance.Category{finance.CategoryGroceries}
	if filter.Matches(tx) {
		t.Error("Should not match groceries category")
	}

	// Test amount filter
	filter = finance.TransactionFilter{
		MinAmount: 50.00,
		MaxAmount: 150.00,
	}
	if !filter.Matches(tx) {
		t.Error("Should match amount range")
	}

	filter.MinAmount = 200.00
	if filter.Matches(tx) {
		t.Error("Should not match high minimum amount")
	}

	t.Log("Transaction filter working correctly")
}

// TestAllCategories tests category list
func TestAllCategories(t *testing.T) {
	categories := finance.AllCategories()

	if len(categories) < 10 {
		t.Errorf("Expected at least 10 categories, got %d", len(categories))
	}

	expectedCategories := []finance.Category{
		finance.CategoryGroceries,
		finance.CategoryDining,
		finance.CategoryTransport,
		finance.CategorySubscription,
		finance.CategoryIncome,
	}

	for _, expected := range expectedCategories {
		found := false
		for _, cat := range categories {
			if cat == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected category: %s", expected)
		}
	}

	t.Logf("All categories: %v", categories)
}

// TestFinanceInsightTypes tests insight creation
func TestFinanceInsightTypes(t *testing.T) {
	types := []finance.InsightType{
		finance.InsightTypeSpendingSummary,
		finance.InsightTypeCategoryBreakdown,
		finance.InsightTypeTrend,
		finance.InsightTypeAnomaly,
		finance.InsightTypeBudgetAlert,
		finance.InsightTypeBillReminder,
		finance.InsightTypeSavingsOpportunity,
	}

	for _, typ := range types {
		if typ == "" {
			t.Error("Insight type should not be empty")
		}
	}

	t.Logf("Insight types: %v", types)
}

// TestTransactionToItem tests converting transaction to item
func TestTransactionToItem(t *testing.T) {
	tx := &finance.CategorizedTransaction{
		Transaction: finance.Transaction{
			TransactionID: "tx-123",
			Name:         "STARBUCKS #456",
			MerchantName: "Starbucks",
			Amount:       5.75,
			Date:         "2024-01-15",
		},
		QLCategory: finance.CategoryDining,
	}

	item := finance.TransactionToItem(tx, "finance-1", core.HatPersonal)

	if item == nil {
		t.Fatal("Failed to convert transaction to item")
	}

	if item.Type != core.ItemTypeTransaction {
		t.Errorf("Expected ItemTypeTransaction, got %s", item.Type)
	}

	if item.HatID != core.HatPersonal {
		t.Errorf("Expected HatPersonal, got %s", item.HatID)
	}

	t.Logf("Transaction item: %s (type: %s)", item.Subject, item.Type)
}
