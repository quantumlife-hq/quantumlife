// Package finance implements transaction categorization.
package finance

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/llm"
)

// Category represents a spending category
type Category string

const (
	CategoryGroceries      Category = "groceries"
	CategoryDining         Category = "dining"
	CategoryTransport      Category = "transport"
	CategoryUtilities      Category = "utilities"
	CategoryEntertainment  Category = "entertainment"
	CategoryShopping       Category = "shopping"
	CategoryHealth         Category = "health"
	CategoryTravel         Category = "travel"
	CategorySubscription   Category = "subscription"
	CategoryBills          Category = "bills"
	CategoryIncome         Category = "income"
	CategoryTransfer       Category = "transfer"
	CategoryInvestment     Category = "investment"
	CategoryFees           Category = "fees"
	CategoryOther          Category = "other"
)

// AllCategories returns all available categories
func AllCategories() []Category {
	return []Category{
		CategoryGroceries,
		CategoryDining,
		CategoryTransport,
		CategoryUtilities,
		CategoryEntertainment,
		CategoryShopping,
		CategoryHealth,
		CategoryTravel,
		CategorySubscription,
		CategoryBills,
		CategoryIncome,
		CategoryTransfer,
		CategoryInvestment,
		CategoryFees,
		CategoryOther,
	}
}

// CategorizedTransaction extends Transaction with our categorization
type CategorizedTransaction struct {
	Transaction
	QLCategory    Category  `json:"ql_category"`
	Subcategory   string    `json:"subcategory,omitempty"`
	IsRecurring   bool      `json:"is_recurring"`
	RecurringID   string    `json:"recurring_id,omitempty"`
	Confidence    float64   `json:"confidence"`
	Tags          []string  `json:"tags,omitempty"`
	CategorizedAt time.Time `json:"categorized_at"`
}

// Categorizer handles transaction categorization
type Categorizer struct {
	llmClient     *llm.OllamaClient
	keywordRules  map[Category][]string
	regexRules    map[Category][]*regexp.Regexp
	merchantCache map[string]Category
}

// CategorizerConfig for the categorizer
type CategorizerConfig struct {
	LLMClient *llm.OllamaClient
	UseAI     bool
}

// NewCategorizer creates a new categorizer
func NewCategorizer(cfg CategorizerConfig) *Categorizer {
	c := &Categorizer{
		llmClient:     cfg.LLMClient,
		merchantCache: make(map[string]Category),
	}
	c.initRules()
	return c
}

// initRules initializes keyword-based categorization rules
func (c *Categorizer) initRules() {
	c.keywordRules = map[Category][]string{
		CategoryGroceries: {
			"walmart", "target", "costco", "kroger", "safeway", "whole foods",
			"trader joe", "aldi", "publix", "wegmans", "giant", "food lion",
			"grocery", "supermarket", "market", "fresh",
		},
		CategoryDining: {
			"mcdonald", "starbucks", "chipotle", "subway", "pizza", "burger",
			"restaurant", "cafe", "coffee", "doordash", "uber eats", "grubhub",
			"postmates", "seamless", "eatery", "grill", "diner", "bistro",
			"kitchen", "sushi", "thai", "chinese", "mexican", "italian",
		},
		CategoryTransport: {
			"uber", "lyft", "taxi", "gas", "shell", "exxon", "chevron", "bp",
			"mobil", "76", "texaco", "parking", "toll", "transit", "metro",
			"bus", "train", "amtrak", "southwest", "delta", "united", "american airlines",
		},
		CategoryUtilities: {
			"electric", "power", "water", "gas bill", "internet", "comcast",
			"verizon", "at&t", "t-mobile", "sprint", "xfinity", "spectrum",
			"utility", "sewage", "trash",
		},
		CategoryEntertainment: {
			"netflix", "hulu", "disney", "hbo", "spotify", "apple music",
			"youtube", "twitch", "movie", "cinema", "theater", "concert",
			"ticketmaster", "stubhub", "gaming", "playstation", "xbox", "steam",
			"audible", "kindle",
		},
		CategoryShopping: {
			"amazon", "ebay", "etsy", "best buy", "apple store", "nike",
			"adidas", "zara", "h&m", "gap", "nordstrom", "macy", "kohls",
			"tjmaxx", "marshalls", "home depot", "lowes", "ikea", "wayfair",
		},
		CategoryHealth: {
			"pharmacy", "cvs", "walgreens", "rite aid", "doctor", "hospital",
			"medical", "dental", "vision", "optometry", "urgent care", "clinic",
			"therapy", "gym", "fitness", "yoga", "crossfit", "planet fitness",
		},
		CategoryTravel: {
			"hotel", "airbnb", "vrbo", "marriott", "hilton", "hyatt",
			"booking.com", "expedia", "kayak", "airline", "flight", "cruise",
			"rental car", "hertz", "enterprise", "avis",
		},
		CategorySubscription: {
			"subscription", "membership", "monthly", "annual", "recurring",
			"patreon", "substack", "medium", "linkedin premium",
		},
		CategoryBills: {
			"rent", "mortgage", "insurance", "geico", "state farm", "allstate",
			"progressive", "loan", "payment", "bill pay",
		},
		CategoryIncome: {
			"payroll", "salary", "deposit", "direct dep", "paycheck",
			"refund", "cashback", "dividend", "interest",
		},
		CategoryTransfer: {
			"transfer", "venmo", "zelle", "paypal", "cash app", "wire",
			"ach", "withdrawal", "atm",
		},
		CategoryInvestment: {
			"robinhood", "fidelity", "vanguard", "schwab", "etrade",
			"coinbase", "crypto", "stock", "invest", "401k", "ira",
		},
		CategoryFees: {
			"fee", "overdraft", "service charge", "atm fee", "foreign transaction",
			"late fee", "annual fee",
		},
	}

	// Compile regex patterns for more complex matching
	c.regexRules = make(map[Category][]*regexp.Regexp)
	for category, keywords := range c.keywordRules {
		for _, keyword := range keywords {
			pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
			c.regexRules[category] = append(c.regexRules[category], pattern)
		}
	}
}

// Categorize categorizes a transaction
func (c *Categorizer) Categorize(tx Transaction) *CategorizedTransaction {
	result := &CategorizedTransaction{
		Transaction:   tx,
		CategorizedAt: time.Now(),
	}

	// Check cache first
	merchantKey := strings.ToLower(tx.MerchantName)
	if merchantKey == "" {
		merchantKey = strings.ToLower(tx.Name)
	}
	if cached, ok := c.merchantCache[merchantKey]; ok {
		result.QLCategory = cached
		result.Confidence = 0.95
		return result
	}

	// Try Plaid's category first
	if len(tx.Category) > 0 {
		result.QLCategory = c.mapPlaidCategory(tx.Category, tx.PersonalFinanceCategory)
		result.Subcategory = strings.Join(tx.Category, " > ")
		result.Confidence = 0.8
		if result.QLCategory != CategoryOther {
			c.merchantCache[merchantKey] = result.QLCategory
			return result
		}
	}

	// Use keyword matching
	category, confidence := c.matchKeywords(tx)
	if category != CategoryOther {
		result.QLCategory = category
		result.Confidence = confidence
		c.merchantCache[merchantKey] = category
		return result
	}

	// Income detection based on amount
	if tx.Amount < 0 { // Plaid uses negative for income
		result.QLCategory = CategoryIncome
		result.Confidence = 0.6
		return result
	}

	result.QLCategory = CategoryOther
	result.Confidence = 0.3
	return result
}

// mapPlaidCategory maps Plaid categories to our categories
func (c *Categorizer) mapPlaidCategory(plaidCategories []string, pfc PersonalFinanceCategory) Category {
	// Use personal finance category if available
	if pfc.Primary != "" {
		switch strings.ToLower(pfc.Primary) {
		case "food_and_drink":
			if strings.Contains(strings.ToLower(pfc.Detailed), "groceries") {
				return CategoryGroceries
			}
			return CategoryDining
		case "transportation":
			return CategoryTransport
		case "travel":
			return CategoryTravel
		case "entertainment":
			return CategoryEntertainment
		case "personal_care":
			return CategoryHealth
		case "medical":
			return CategoryHealth
		case "general_merchandise":
			return CategoryShopping
		case "income":
			return CategoryIncome
		case "transfer_out", "transfer_in":
			return CategoryTransfer
		case "loan_payments":
			return CategoryBills
		case "rent_and_utilities":
			return CategoryUtilities
		}
	}

	// Fallback to legacy categories
	if len(plaidCategories) == 0 {
		return CategoryOther
	}

	primary := strings.ToLower(plaidCategories[0])
	switch primary {
	case "food and drink":
		if len(plaidCategories) > 1 && strings.ToLower(plaidCategories[1]) == "groceries" {
			return CategoryGroceries
		}
		return CategoryDining
	case "travel":
		return CategoryTravel
	case "transportation":
		return CategoryTransport
	case "shops":
		return CategoryShopping
	case "recreation":
		return CategoryEntertainment
	case "healthcare":
		return CategoryHealth
	case "service":
		if len(plaidCategories) > 1 {
			sub := strings.ToLower(plaidCategories[1])
			if strings.Contains(sub, "utilities") {
				return CategoryUtilities
			}
			if strings.Contains(sub, "insurance") {
				return CategoryBills
			}
			if strings.Contains(sub, "subscription") {
				return CategorySubscription
			}
		}
	case "payment":
		return CategoryBills
	case "transfer":
		return CategoryTransfer
	case "bank fees":
		return CategoryFees
	}

	return CategoryOther
}

// matchKeywords matches transaction against keyword rules
func (c *Categorizer) matchKeywords(tx Transaction) (Category, float64) {
	text := strings.ToLower(tx.MerchantName + " " + tx.Name)

	var bestCategory Category = CategoryOther
	var bestScore float64 = 0

	for category, patterns := range c.regexRules {
		for _, pattern := range patterns {
			if pattern.MatchString(text) {
				// Weight by specificity (longer matches are more specific)
				match := pattern.FindString(text)
				score := float64(len(match)) / float64(len(text)+1)
				score = 0.5 + score*0.4 // Base confidence + weighted

				if score > bestScore {
					bestScore = score
					bestCategory = category
				}
			}
		}
	}

	return bestCategory, bestScore
}

// CategorizeWithAI uses LLM for uncertain transactions
func (c *Categorizer) CategorizeWithAI(ctx context.Context, tx Transaction) (*CategorizedTransaction, error) {
	result := c.Categorize(tx)

	// Only use AI for low confidence categorizations
	if result.Confidence >= 0.7 || c.llmClient == nil {
		return result, nil
	}

	system := "You are a transaction categorizer. Respond with only the category name, nothing else."
	prompt := `Categorize this transaction into exactly one of these categories:
groceries, dining, transport, utilities, entertainment, shopping, health, travel, subscription, bills, income, transfer, investment, fees, other

Transaction:
- Name: ` + tx.Name + `
- Merchant: ` + tx.MerchantName + `
- Amount: $` + formatAmount(tx.Amount) + `
- Original categories: ` + strings.Join(tx.Category, ", ") + `

Respond with only the category name, nothing else.`

	resp, err := c.llmClient.Chat(ctx, system, prompt)
	if err != nil {
		return result, nil // Fall back to keyword categorization
	}

	aiCategory := Category(strings.TrimSpace(strings.ToLower(resp)))

	// Validate the category
	for _, valid := range AllCategories() {
		if aiCategory == valid {
			result.QLCategory = aiCategory
			result.Confidence = 0.85
			result.Tags = append(result.Tags, "ai_categorized")
			break
		}
	}

	return result, nil
}

// BatchCategorize categorizes multiple transactions
func (c *Categorizer) BatchCategorize(transactions []Transaction) []*CategorizedTransaction {
	results := make([]*CategorizedTransaction, len(transactions))
	for i, tx := range transactions {
		results[i] = c.Categorize(tx)
	}
	return results
}

// RecurringDetector finds recurring transactions
type RecurringDetector struct {
	minOccurrences int
	varianceDays   int
}

// NewRecurringDetector creates a new recurring detector
func NewRecurringDetector() *RecurringDetector {
	return &RecurringDetector{
		minOccurrences: 2,
		varianceDays:   5, // Allow +/- 5 days variance
	}
}

// RecurringTransaction represents a detected recurring payment
type RecurringTransaction struct {
	ID           string     `json:"id"`
	MerchantName string     `json:"merchant_name"`
	Category     Category   `json:"category"`
	Amount       float64    `json:"amount"`
	Frequency    string     `json:"frequency"` // weekly, biweekly, monthly, annual
	DayOfMonth   int        `json:"day_of_month,omitempty"`
	Transactions []string   `json:"transaction_ids"`
	NextExpected time.Time  `json:"next_expected"`
	LastSeen     time.Time  `json:"last_seen"`
	IsActive     bool       `json:"is_active"`
}

// DetectRecurring identifies recurring transactions
func (d *RecurringDetector) DetectRecurring(transactions []*CategorizedTransaction) []*RecurringTransaction {
	// Group by merchant/name
	byMerchant := make(map[string][]*CategorizedTransaction)
	for _, tx := range transactions {
		key := strings.ToLower(tx.MerchantName)
		if key == "" {
			key = strings.ToLower(tx.Name)
		}
		byMerchant[key] = append(byMerchant[key], tx)
	}

	var recurring []*RecurringTransaction

	for merchant, txs := range byMerchant {
		if len(txs) < d.minOccurrences {
			continue
		}

		// Sort by date
		sort.Slice(txs, func(i, j int) bool {
			return txs[i].Date < txs[j].Date
		})

		// Group by similar amounts (within 10%)
		amountGroups := d.groupByAmount(txs)

		for _, group := range amountGroups {
			if len(group) < d.minOccurrences {
				continue
			}

			// Detect frequency
			freq, dayOfMonth := d.detectFrequency(group)
			if freq == "" {
				continue
			}

			ids := make([]string, len(group))
			for i, tx := range group {
				ids[i] = tx.TransactionID
				tx.IsRecurring = true
			}

			lastDate, _ := time.Parse("2006-01-02", group[len(group)-1].Date)
			nextExpected := d.predictNext(lastDate, freq, dayOfMonth)

			rec := &RecurringTransaction{
				ID:           "rec_" + merchant[:min(8, len(merchant))],
				MerchantName: txs[0].MerchantName,
				Category:     group[0].QLCategory,
				Amount:       d.averageAmount(group),
				Frequency:    freq,
				DayOfMonth:   dayOfMonth,
				Transactions: ids,
				NextExpected: nextExpected,
				LastSeen:     lastDate,
				IsActive:     time.Since(lastDate) < 45*24*time.Hour,
			}
			recurring = append(recurring, rec)
		}
	}

	return recurring
}

// groupByAmount groups transactions with similar amounts
func (d *RecurringDetector) groupByAmount(txs []*CategorizedTransaction) [][]*CategorizedTransaction {
	if len(txs) == 0 {
		return nil
	}

	var groups [][]*CategorizedTransaction
	used := make(map[int]bool)

	for i, tx := range txs {
		if used[i] {
			continue
		}

		group := []*CategorizedTransaction{tx}
		used[i] = true

		for j := i + 1; j < len(txs); j++ {
			if used[j] {
				continue
			}

			// Check if amounts are similar (within 10%)
			ratio := tx.Amount / txs[j].Amount
			if ratio >= 0.9 && ratio <= 1.1 {
				group = append(group, txs[j])
				used[j] = true
			}
		}

		groups = append(groups, group)
	}

	return groups
}

// detectFrequency detects the payment frequency
func (d *RecurringDetector) detectFrequency(txs []*CategorizedTransaction) (string, int) {
	if len(txs) < 2 {
		return "", 0
	}

	// Calculate intervals between transactions
	var intervals []int
	for i := 1; i < len(txs); i++ {
		date1, _ := time.Parse("2006-01-02", txs[i-1].Date)
		date2, _ := time.Parse("2006-01-02", txs[i].Date)
		days := int(date2.Sub(date1).Hours() / 24)
		intervals = append(intervals, days)
	}

	avgInterval := average(intervals)

	// Determine frequency based on average interval
	switch {
	case avgInterval >= 6 && avgInterval <= 8:
		return "weekly", 0
	case avgInterval >= 13 && avgInterval <= 16:
		return "biweekly", 0
	case avgInterval >= 27 && avgInterval <= 35:
		// Monthly - determine typical day
		dayOfMonth := d.detectDayOfMonth(txs)
		return "monthly", dayOfMonth
	case avgInterval >= 85 && avgInterval <= 100:
		return "quarterly", 0
	case avgInterval >= 355 && avgInterval <= 375:
		return "annual", 0
	default:
		return "", 0
	}
}

// detectDayOfMonth finds the typical day of month for payments
func (d *RecurringDetector) detectDayOfMonth(txs []*CategorizedTransaction) int {
	days := make(map[int]int)
	for _, tx := range txs {
		date, _ := time.Parse("2006-01-02", tx.Date)
		days[date.Day()]++
	}

	maxCount := 0
	typicalDay := 0
	for day, count := range days {
		if count > maxCount {
			maxCount = count
			typicalDay = day
		}
	}
	return typicalDay
}

// predictNext predicts the next occurrence
func (d *RecurringDetector) predictNext(lastDate time.Time, frequency string, dayOfMonth int) time.Time {
	switch frequency {
	case "weekly":
		return lastDate.AddDate(0, 0, 7)
	case "biweekly":
		return lastDate.AddDate(0, 0, 14)
	case "monthly":
		next := lastDate.AddDate(0, 1, 0)
		if dayOfMonth > 0 {
			next = time.Date(next.Year(), next.Month(), dayOfMonth, 0, 0, 0, 0, next.Location())
		}
		return next
	case "quarterly":
		return lastDate.AddDate(0, 3, 0)
	case "annual":
		return lastDate.AddDate(1, 0, 0)
	default:
		return lastDate.AddDate(0, 1, 0)
	}
}

// averageAmount calculates average transaction amount
func (d *RecurringDetector) averageAmount(txs []*CategorizedTransaction) float64 {
	if len(txs) == 0 {
		return 0
	}
	var sum float64
	for _, tx := range txs {
		sum += tx.Amount
	}
	return sum / float64(len(txs))
}

// Helper functions
func average(nums []int) float64 {
	if len(nums) == 0 {
		return 0
	}
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}

func formatAmount(amount float64) string {
	if amount < 0 {
		amount = -amount
	}
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					fmt.Sprintf("%.2f", amount),
					".", ",", 1),
				",00", "", 1),
			",", ".", 1),
		"0"), ".")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
