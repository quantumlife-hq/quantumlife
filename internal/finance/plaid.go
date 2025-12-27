// Package finance implements Plaid integration for banking.
package finance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Environment represents the Plaid environment
type Environment string

const (
	EnvironmentSandbox     Environment = "sandbox"
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

// PlaidConfig holds Plaid API configuration
type PlaidConfig struct {
	ClientID     string
	Secret       string
	Environment  Environment
	ClientName   string
	CountryCodes []string
	Products     []string
	Language     string
}

// DefaultPlaidConfig returns sandbox configuration
func DefaultPlaidConfig() PlaidConfig {
	return PlaidConfig{
		ClientID:     os.Getenv("PLAID_CLIENT_ID"),
		Secret:       os.Getenv("PLAID_SECRET"),
		Environment:  EnvironmentSandbox,
		ClientName:   "QuantumLife",
		CountryCodes: []string{"US"},
		Products:     []string{"transactions"},
		Language:     "en",
	}
}

// IsConfigured checks if Plaid credentials are set
func IsConfigured() bool {
	return os.Getenv("PLAID_CLIENT_ID") != "" && os.Getenv("PLAID_SECRET") != ""
}

// PlaidClient handles Plaid API interactions
type PlaidClient struct {
	config     PlaidConfig
	httpClient *http.Client
	baseURL    string
}

// NewPlaidClient creates a new Plaid client
func NewPlaidClient(cfg PlaidConfig) *PlaidClient {
	var baseURL string
	switch cfg.Environment {
	case EnvironmentSandbox:
		baseURL = "https://sandbox.plaid.com"
	case EnvironmentDevelopment:
		baseURL = "https://development.plaid.com"
	case EnvironmentProduction:
		baseURL = "https://production.plaid.com"
	default:
		baseURL = "https://sandbox.plaid.com"
	}

	return &PlaidClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// request makes a request to the Plaid API
func (c *PlaidClient) request(ctx context.Context, endpoint string, body interface{}, result interface{}) error {
	// Add credentials to body
	bodyMap := make(map[string]interface{})
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
			return fmt.Errorf("unmarshal body: %w", err)
		}
	}
	bodyMap["client_id"] = c.config.ClientID
	bodyMap["secret"] = c.config.Secret

	jsonBody, err := json.Marshal(bodyMap)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var plaidErr PlaidError
		if err := json.Unmarshal(respBody, &plaidErr); err == nil {
			return &plaidErr
		}
		return fmt.Errorf("plaid error: %s", string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}

// PlaidError represents a Plaid API error
type PlaidError struct {
	ErrorType    string `json:"error_type"`
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	DisplayMsg   string `json:"display_message"`
	RequestID    string `json:"request_id"`
}

func (e *PlaidError) Error() string {
	return fmt.Sprintf("plaid: %s - %s", e.ErrorCode, e.ErrorMessage)
}

// LinkTokenResponse from /link/token/create
type LinkTokenResponse struct {
	LinkToken  string    `json:"link_token"`
	Expiration time.Time `json:"expiration"`
	RequestID  string    `json:"request_id"`
}

// CreateLinkToken creates a link token for Plaid Link
func (c *PlaidClient) CreateLinkToken(ctx context.Context, userID string) (*LinkTokenResponse, error) {
	req := map[string]interface{}{
		"user": map[string]string{
			"client_user_id": userID,
		},
		"client_name":   c.config.ClientName,
		"products":      c.config.Products,
		"country_codes": c.config.CountryCodes,
		"language":      c.config.Language,
	}

	var resp LinkTokenResponse
	if err := c.request(ctx, "/link/token/create", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ExchangeTokenResponse from /item/public_token/exchange
type ExchangeTokenResponse struct {
	AccessToken string `json:"access_token"`
	ItemID      string `json:"item_id"`
	RequestID   string `json:"request_id"`
}

// ExchangePublicToken exchanges a public token for an access token
func (c *PlaidClient) ExchangePublicToken(ctx context.Context, publicToken string) (*ExchangeTokenResponse, error) {
	req := map[string]string{
		"public_token": publicToken,
	}

	var resp ExchangeTokenResponse
	if err := c.request(ctx, "/item/public_token/exchange", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Account represents a Plaid account
type Account struct {
	AccountID    string         `json:"account_id"`
	Name         string         `json:"name"`
	OfficialName string         `json:"official_name"`
	Type         string         `json:"type"`     // depository, credit, loan, investment
	Subtype      string         `json:"subtype"`  // checking, savings, credit card, etc.
	Mask         string         `json:"mask"`     // Last 4 digits
	Balances     AccountBalance `json:"balances"`
}

// AccountBalance represents account balance info
type AccountBalance struct {
	Current              float64 `json:"current"`
	Available            float64 `json:"available"`
	Limit                float64 `json:"limit"`
	IsoCurrencyCode      string  `json:"iso_currency_code"`
	UnofficialCurrencyCode string `json:"unofficial_currency_code"`
}

// AccountsResponse from /accounts/get
type AccountsResponse struct {
	Accounts  []Account `json:"accounts"`
	Item      Item      `json:"item"`
	RequestID string    `json:"request_id"`
}

// Item represents a Plaid Item (bank connection)
type Item struct {
	ItemID            string   `json:"item_id"`
	InstitutionID     string   `json:"institution_id"`
	AvailableProducts []string `json:"available_products"`
	Products          []string `json:"products"`
	ConsentExpiration string   `json:"consent_expiration_time"`
}

// GetAccounts retrieves accounts for an access token
func (c *PlaidClient) GetAccounts(ctx context.Context, accessToken string) (*AccountsResponse, error) {
	req := map[string]string{
		"access_token": accessToken,
	}

	var resp AccountsResponse
	if err := c.request(ctx, "/accounts/get", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Transaction represents a Plaid transaction
type Transaction struct {
	TransactionID     string   `json:"transaction_id"`
	AccountID         string   `json:"account_id"`
	Amount            float64  `json:"amount"`
	Date              string   `json:"date"`
	AuthorizedDate    string   `json:"authorized_date"`
	Name              string   `json:"name"`
	MerchantName      string   `json:"merchant_name"`
	Category          []string `json:"category"`
	CategoryID        string   `json:"category_id"`
	PaymentChannel    string   `json:"payment_channel"` // online, in store, other
	Pending           bool     `json:"pending"`
	IsoCurrencyCode   string   `json:"iso_currency_code"`
	Location          Location `json:"location"`
	PersonalFinanceCategory PersonalFinanceCategory `json:"personal_finance_category"`
}

// Location contains transaction location info
type Location struct {
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Region      string  `json:"region"`
	PostalCode  string  `json:"postal_code"`
	Country     string  `json:"country"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	StoreNumber string  `json:"store_number"`
}

// PersonalFinanceCategory from Plaid's enriched categorization
type PersonalFinanceCategory struct {
	Primary   string `json:"primary"`
	Detailed  string `json:"detailed"`
}

// TransactionsResponse from /transactions/get
type TransactionsResponse struct {
	Accounts          []Account     `json:"accounts"`
	Transactions      []Transaction `json:"transactions"`
	TotalTransactions int           `json:"total_transactions"`
	Item              Item          `json:"item"`
	RequestID         string        `json:"request_id"`
}

// GetTransactions retrieves transactions for a date range
func (c *PlaidClient) GetTransactions(ctx context.Context, accessToken string, startDate, endDate time.Time) (*TransactionsResponse, error) {
	req := map[string]interface{}{
		"access_token": accessToken,
		"start_date":   startDate.Format("2006-01-02"),
		"end_date":     endDate.Format("2006-01-02"),
		"options": map[string]interface{}{
			"count":                      500,
			"offset":                     0,
			"include_personal_finance_category": true,
		},
	}

	var resp TransactionsResponse
	if err := c.request(ctx, "/transactions/get", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetAllTransactions retrieves all transactions with pagination
func (c *PlaidClient) GetAllTransactions(ctx context.Context, accessToken string, startDate, endDate time.Time) ([]Transaction, error) {
	var allTransactions []Transaction
	offset := 0
	count := 500

	for {
		req := map[string]interface{}{
			"access_token": accessToken,
			"start_date":   startDate.Format("2006-01-02"),
			"end_date":     endDate.Format("2006-01-02"),
			"options": map[string]interface{}{
				"count":                      count,
				"offset":                     offset,
				"include_personal_finance_category": true,
			},
		}

		var resp TransactionsResponse
		if err := c.request(ctx, "/transactions/get", req, &resp); err != nil {
			return nil, err
		}

		allTransactions = append(allTransactions, resp.Transactions...)

		if len(resp.Transactions) < count || len(allTransactions) >= resp.TotalTransactions {
			break
		}

		offset += count
	}

	return allTransactions, nil
}

// TransactionsSyncResponse from /transactions/sync
type TransactionsSyncResponse struct {
	Added       []Transaction `json:"added"`
	Modified    []Transaction `json:"modified"`
	Removed     []RemovedTransaction `json:"removed"`
	NextCursor  string        `json:"next_cursor"`
	HasMore     bool          `json:"has_more"`
	RequestID   string        `json:"request_id"`
}

// RemovedTransaction represents a removed transaction
type RemovedTransaction struct {
	TransactionID string `json:"transaction_id"`
}

// SyncTransactions uses the transactions sync endpoint
func (c *PlaidClient) SyncTransactions(ctx context.Context, accessToken, cursor string) (*TransactionsSyncResponse, error) {
	req := map[string]interface{}{
		"access_token": accessToken,
		"options": map[string]interface{}{
			"include_personal_finance_category": true,
		},
	}
	if cursor != "" {
		req["cursor"] = cursor
	}

	var resp TransactionsSyncResponse
	if err := c.request(ctx, "/transactions/sync", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Institution represents a financial institution
type Institution struct {
	InstitutionID string   `json:"institution_id"`
	Name          string   `json:"name"`
	Products      []string `json:"products"`
	CountryCodes  []string `json:"country_codes"`
	URL           string   `json:"url"`
	Logo          string   `json:"logo"`
	PrimaryColor  string   `json:"primary_color"`
}

// GetInstitution retrieves institution details
func (c *PlaidClient) GetInstitution(ctx context.Context, institutionID string) (*Institution, error) {
	req := map[string]interface{}{
		"institution_id": institutionID,
		"country_codes":  c.config.CountryCodes,
		"options": map[string]bool{
			"include_optional_metadata": true,
		},
	}

	var resp struct {
		Institution Institution `json:"institution"`
		RequestID   string      `json:"request_id"`
	}
	if err := c.request(ctx, "/institutions/get_by_id", req, &resp); err != nil {
		return nil, err
	}

	return &resp.Institution, nil
}

// Balance retrieves current balances
func (c *PlaidClient) Balance(ctx context.Context, accessToken string) (*AccountsResponse, error) {
	req := map[string]string{
		"access_token": accessToken,
	}

	var resp AccountsResponse
	if err := c.request(ctx, "/accounts/balance/get", req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// RemoveItem removes a connected institution
func (c *PlaidClient) RemoveItem(ctx context.Context, accessToken string) error {
	req := map[string]string{
		"access_token": accessToken,
	}

	return c.request(ctx, "/item/remove", req, nil)
}

// CreateSandboxPublicToken creates a test public token (sandbox only)
func (c *PlaidClient) CreateSandboxPublicToken(ctx context.Context, institutionID string, products []string) (string, error) {
	if c.config.Environment != EnvironmentSandbox {
		return "", fmt.Errorf("sandbox only endpoint")
	}

	req := map[string]interface{}{
		"institution_id":   institutionID,
		"initial_products": products,
	}

	var resp struct {
		PublicToken string `json:"public_token"`
		RequestID   string `json:"request_id"`
	}
	if err := c.request(ctx, "/sandbox/public_token/create", req, &resp); err != nil {
		return "", err
	}

	return resp.PublicToken, nil
}

// FireWebhook fires a test webhook (sandbox only)
func (c *PlaidClient) FireWebhook(ctx context.Context, accessToken, webhookCode string) error {
	if c.config.Environment != EnvironmentSandbox {
		return fmt.Errorf("sandbox only endpoint")
	}

	req := map[string]string{
		"access_token": accessToken,
		"webhook_code": webhookCode,
	}

	return c.request(ctx, "/sandbox/item/fire_webhook", req, nil)
}

// Connection represents a stored bank connection
type Connection struct {
	ID            string    `json:"id"`
	ItemID        string    `json:"item_id"`
	InstitutionID string    `json:"institution_id"`
	InstitutionName string  `json:"institution_name"`
	AccessToken   string    `json:"access_token"` // Encrypted in storage
	UserID        string    `json:"user_id"`
	Status        string    `json:"status"` // active, error, pending
	Accounts      []Account `json:"accounts"`
	LastSync      time.Time `json:"last_sync"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	SyncCursor    string    `json:"sync_cursor"`
}

// ConnectionStatus values
const (
	ConnectionStatusActive  = "active"
	ConnectionStatusError   = "error"
	ConnectionStatusPending = "pending"
)
