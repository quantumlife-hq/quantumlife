// Package finance implements financial insights and alerts.
package finance

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/quantumlife/quantumlife/internal/llm"
)

// InsightType represents the type of financial insight
type InsightType string

const (
	InsightTypeSpendingSummary   InsightType = "spending_summary"
	InsightTypeCategoryBreakdown InsightType = "category_breakdown"
	InsightTypeTrend             InsightType = "trend"
	InsightTypeAnomaly           InsightType = "anomaly"
	InsightTypeBudgetAlert       InsightType = "budget_alert"
	InsightTypeBillReminder      InsightType = "bill_reminder"
	InsightTypeSavingsOpportunity InsightType = "savings_opportunity"
	InsightTypeIncomeAnalysis    InsightType = "income_analysis"
)

// Severity levels for alerts
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityAlert   Severity = "alert"
)

// Insight represents a financial insight
type Insight struct {
	ID          string      `json:"id"`
	Type        InsightType `json:"type"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Severity    Severity    `json:"severity"`
	Amount      float64     `json:"amount,omitempty"`
	Category    Category    `json:"category,omitempty"`
	Period      string      `json:"period,omitempty"`
	Trend       string      `json:"trend,omitempty"` // up, down, stable
	Data        interface{} `json:"data,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at,omitempty"`
}

// SpendingSummary holds spending analysis
type SpendingSummary struct {
	Period       string              `json:"period"`
	TotalSpent   float64             `json:"total_spent"`
	TotalIncome  float64             `json:"total_income"`
	NetCashFlow  float64             `json:"net_cash_flow"`
	ByCategory   map[Category]float64 `json:"by_category"`
	TopMerchants []MerchantSpend     `json:"top_merchants"`
	DailyAverage float64             `json:"daily_average"`
	Transactions int                 `json:"transactions"`
}

// MerchantSpend tracks spending per merchant
type MerchantSpend struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
	Count  int     `json:"count"`
}

// Alert represents a financial alert
type Alert struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Severity    Severity  `json:"severity"`
	ActionURL   string    `json:"action_url,omitempty"`
	Dismissed   bool      `json:"dismissed"`
	CreatedAt   time.Time `json:"created_at"`
	TriggeredBy string    `json:"triggered_by,omitempty"`
}

// InsightsEngine generates financial insights
type InsightsEngine struct {
	llmClient     *llm.OllamaClient
	budgets       map[Category]float64
	alertRules    []AlertRule
}

// AlertRule defines when to trigger an alert
type AlertRule struct {
	Name      string
	Condition func(txs []*CategorizedTransaction) *Alert
}

// InsightsConfig for the engine
type InsightsConfig struct {
	LLMClient *llm.OllamaClient
	Budgets   map[Category]float64
}

// NewInsightsEngine creates a new insights engine
func NewInsightsEngine(cfg InsightsConfig) *InsightsEngine {
	e := &InsightsEngine{
		llmClient: cfg.LLMClient,
		budgets:   cfg.Budgets,
	}
	if e.budgets == nil {
		e.budgets = make(map[Category]float64)
	}
	e.initAlertRules()
	return e
}

// SetBudget sets a budget for a category
func (e *InsightsEngine) SetBudget(category Category, amount float64) {
	e.budgets[category] = amount
}

// GetBudget returns the budget for a category
func (e *InsightsEngine) GetBudget(category Category) (float64, bool) {
	amount, ok := e.budgets[category]
	return amount, ok
}

// GenerateSpendingSummary creates a spending summary
func (e *InsightsEngine) GenerateSpendingSummary(transactions []*CategorizedTransaction, period string) *SpendingSummary {
	summary := &SpendingSummary{
		Period:       period,
		ByCategory:   make(map[Category]float64),
		Transactions: len(transactions),
	}

	merchantSpend := make(map[string]*MerchantSpend)

	for _, tx := range transactions {
		amount := tx.Amount

		if amount < 0 {
			// Income (Plaid uses negative for credits)
			summary.TotalIncome += -amount
		} else {
			// Expense
			summary.TotalSpent += amount
			summary.ByCategory[tx.QLCategory] += amount

			// Track merchant
			merchant := tx.MerchantName
			if merchant == "" {
				merchant = tx.Name
			}
			if ms, ok := merchantSpend[merchant]; ok {
				ms.Amount += amount
				ms.Count++
			} else {
				merchantSpend[merchant] = &MerchantSpend{
					Name:   merchant,
					Amount: amount,
					Count:  1,
				}
			}
		}
	}

	summary.NetCashFlow = summary.TotalIncome - summary.TotalSpent

	// Calculate daily average based on period
	days := e.periodToDays(period)
	if days > 0 {
		summary.DailyAverage = summary.TotalSpent / float64(days)
	}

	// Top merchants
	for _, ms := range merchantSpend {
		summary.TopMerchants = append(summary.TopMerchants, *ms)
	}
	sort.Slice(summary.TopMerchants, func(i, j int) bool {
		return summary.TopMerchants[i].Amount > summary.TopMerchants[j].Amount
	})
	if len(summary.TopMerchants) > 10 {
		summary.TopMerchants = summary.TopMerchants[:10]
	}

	return summary
}

// periodToDays converts period string to days
func (e *InsightsEngine) periodToDays(period string) int {
	switch period {
	case "week":
		return 7
	case "month":
		return 30
	case "quarter":
		return 90
	case "year":
		return 365
	default:
		return 30
	}
}

// GenerateCategoryInsights creates category-specific insights
func (e *InsightsEngine) GenerateCategoryInsights(transactions []*CategorizedTransaction) []*Insight {
	var insights []*Insight

	// Group by category
	byCategory := make(map[Category][]*CategorizedTransaction)
	for _, tx := range transactions {
		if tx.Amount > 0 { // Only expenses
			byCategory[tx.QLCategory] = append(byCategory[tx.QLCategory], tx)
		}
	}

	// Generate insights for each category
	for category, txs := range byCategory {
		var total float64
		for _, tx := range txs {
			total += tx.Amount
		}

		// Check against budget
		if budget, ok := e.budgets[category]; ok {
			percentUsed := (total / budget) * 100

			if percentUsed >= 100 {
				insights = append(insights, &Insight{
					ID:          fmt.Sprintf("budget_over_%s", category),
					Type:        InsightTypeBudgetAlert,
					Title:       fmt.Sprintf("%s Budget Exceeded", capitalizeCategory(category)),
					Description: fmt.Sprintf("You've spent $%.2f on %s, exceeding your $%.2f budget by $%.2f", total, category, budget, total-budget),
					Severity:    SeverityAlert,
					Amount:      total,
					Category:    category,
					CreatedAt:   time.Now(),
				})
			} else if percentUsed >= 80 {
				insights = append(insights, &Insight{
					ID:          fmt.Sprintf("budget_warning_%s", category),
					Type:        InsightTypeBudgetAlert,
					Title:       fmt.Sprintf("%s Budget Warning", capitalizeCategory(category)),
					Description: fmt.Sprintf("You've used %.0f%% of your %s budget ($%.2f of $%.2f)", percentUsed, category, total, budget),
					Severity:    SeverityWarning,
					Amount:      total,
					Category:    category,
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	return insights
}

// DetectAnomalies finds unusual spending patterns
func (e *InsightsEngine) DetectAnomalies(transactions []*CategorizedTransaction) []*Insight {
	var insights []*Insight

	// Group by category
	byCategory := make(map[Category][]float64)
	for _, tx := range transactions {
		if tx.Amount > 0 {
			byCategory[tx.QLCategory] = append(byCategory[tx.QLCategory], tx.Amount)
		}
	}

	// Find anomalies (transactions > 2x the average for that category)
	for category, amounts := range byCategory {
		if len(amounts) < 3 {
			continue
		}

		avg := sum(amounts) / float64(len(amounts))
		threshold := avg * 2

		for i, tx := range transactions {
			if tx.QLCategory == category && tx.Amount > threshold && tx.Amount > 100 {
				insights = append(insights, &Insight{
					ID:          fmt.Sprintf("anomaly_%d", i),
					Type:        InsightTypeAnomaly,
					Title:       "Unusual Transaction Detected",
					Description: fmt.Sprintf("$%.2f at %s is %.1fx higher than your usual %s spending", tx.Amount, tx.MerchantName, tx.Amount/avg, category),
					Severity:    SeverityInfo,
					Amount:      tx.Amount,
					Category:    category,
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	return insights
}

// GenerateBillReminders creates reminders for upcoming bills
func (e *InsightsEngine) GenerateBillReminders(recurring []*RecurringTransaction) []*Insight {
	var insights []*Insight
	now := time.Now()

	for _, rec := range recurring {
		if !rec.IsActive {
			continue
		}

		daysUntil := int(rec.NextExpected.Sub(now).Hours() / 24)

		if daysUntil <= 3 && daysUntil >= 0 {
			insights = append(insights, &Insight{
				ID:          fmt.Sprintf("bill_%s", rec.ID),
				Type:        InsightTypeBillReminder,
				Title:       "Upcoming Bill",
				Description: fmt.Sprintf("%s payment of $%.2f is due %s", rec.MerchantName, rec.Amount, formatDaysUntil(daysUntil)),
				Severity:    SeverityInfo,
				Amount:      rec.Amount,
				Category:    rec.Category,
				CreatedAt:   now,
				ExpiresAt:   rec.NextExpected,
				Data:        rec,
			})
		}
	}

	return insights
}

// FindSavingsOpportunities identifies potential savings
func (e *InsightsEngine) FindSavingsOpportunities(transactions []*CategorizedTransaction, recurring []*RecurringTransaction) []*Insight {
	var insights []*Insight

	// Find overlapping subscriptions
	subscriptionCategories := []Category{CategorySubscription, CategoryEntertainment}
	var subs []*RecurringTransaction
	for _, rec := range recurring {
		for _, cat := range subscriptionCategories {
			if rec.Category == cat {
				subs = append(subs, rec)
				break
			}
		}
	}

	if len(subs) >= 3 {
		var monthlyTotal float64
		for _, sub := range subs {
			switch sub.Frequency {
			case "weekly":
				monthlyTotal += sub.Amount * 4.33
			case "biweekly":
				monthlyTotal += sub.Amount * 2.17
			case "monthly":
				monthlyTotal += sub.Amount
			case "annual":
				monthlyTotal += sub.Amount / 12
			}
		}

		insights = append(insights, &Insight{
			ID:          "subscriptions_review",
			Type:        InsightTypeSavingsOpportunity,
			Title:       "Subscription Review",
			Description: fmt.Sprintf("You have %d active subscriptions totaling $%.2f/month. Review to find savings.", len(subs), monthlyTotal),
			Severity:    SeverityInfo,
			Amount:      monthlyTotal,
			CreatedAt:   time.Now(),
			Data:        subs,
		})
	}

	// Find high dining spending vs groceries
	summary := e.GenerateSpendingSummary(transactions, "month")
	dining := summary.ByCategory[CategoryDining]
	groceries := summary.ByCategory[CategoryGroceries]

	if dining > groceries && dining > 200 {
		insights = append(insights, &Insight{
			ID:          "dining_vs_groceries",
			Type:        InsightTypeSavingsOpportunity,
			Title:       "Dining Expenses High",
			Description: fmt.Sprintf("You spent $%.2f dining out vs $%.2f on groceries. Cooking more could save money.", dining, groceries),
			Severity:    SeverityInfo,
			Amount:      dining - groceries,
			CreatedAt:   time.Now(),
		})
	}

	return insights
}

// GenerateAIInsights uses LLM for deeper analysis
func (e *InsightsEngine) GenerateAIInsights(ctx context.Context, summary *SpendingSummary) (*Insight, error) {
	if e.llmClient == nil {
		return nil, fmt.Errorf("no LLM client configured")
	}

	// Build prompt
	categoryBreakdown := ""
	for cat, amount := range summary.ByCategory {
		if amount > 0 {
			categoryBreakdown += fmt.Sprintf("- %s: $%.2f\n", cat, amount)
		}
	}

	prompt := fmt.Sprintf(`Analyze this financial summary and provide one key insight or recommendation:

Period: %s
Total Spent: $%.2f
Total Income: $%.2f
Net Cash Flow: $%.2f
Daily Average: $%.2f

Category Breakdown:
%s

Top Merchants:
%s

Provide a brief, actionable insight (2-3 sentences max).`,
		summary.Period,
		summary.TotalSpent,
		summary.TotalIncome,
		summary.NetCashFlow,
		summary.DailyAverage,
		categoryBreakdown,
		formatMerchants(summary.TopMerchants[:min(5, len(summary.TopMerchants))]),
	)

	system := "You are a helpful financial advisor. Provide brief, actionable insights."
	resp, err := e.llmClient.Chat(ctx, system, prompt)
	if err != nil {
		return nil, err
	}

	return &Insight{
		ID:          fmt.Sprintf("ai_insight_%d", time.Now().Unix()),
		Type:        InsightTypeTrend,
		Title:       "AI Financial Insight",
		Description: resp,
		Severity:    SeverityInfo,
		Period:      summary.Period,
		CreatedAt:   time.Now(),
		Data:        map[string]string{"source": "ai_analysis"},
	}, nil
}

// initAlertRules sets up default alert rules
func (e *InsightsEngine) initAlertRules() {
	e.alertRules = []AlertRule{
		{
			Name: "Large Transaction",
			Condition: func(txs []*CategorizedTransaction) *Alert {
				for _, tx := range txs {
					if tx.Amount > 500 && tx.Amount > 0 {
						return &Alert{
							ID:          fmt.Sprintf("large_%s", tx.TransactionID),
							Type:        "large_transaction",
							Title:       "Large Transaction Alert",
							Message:     fmt.Sprintf("$%.2f transaction at %s", tx.Amount, tx.MerchantName),
							Severity:    SeverityWarning,
							CreatedAt:   time.Now(),
							TriggeredBy: tx.TransactionID,
						}
					}
				}
				return nil
			},
		},
		{
			Name: "Fee Detected",
			Condition: func(txs []*CategorizedTransaction) *Alert {
				for _, tx := range txs {
					if tx.QLCategory == CategoryFees && tx.Amount > 0 {
						return &Alert{
							ID:          fmt.Sprintf("fee_%s", tx.TransactionID),
							Type:        "fee_detected",
							Title:       "Bank Fee Detected",
							Message:     fmt.Sprintf("$%.2f fee charged: %s", tx.Amount, tx.Name),
							Severity:    SeverityWarning,
							CreatedAt:   time.Now(),
							TriggeredBy: tx.TransactionID,
						}
					}
				}
				return nil
			},
		},
	}
}

// CheckAlerts runs alert rules against transactions
func (e *InsightsEngine) CheckAlerts(transactions []*CategorizedTransaction) []*Alert {
	var alerts []*Alert
	for _, rule := range e.alertRules {
		if alert := rule.Condition(transactions); alert != nil {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// Helper functions
func sum(nums []float64) float64 {
	var total float64
	for _, n := range nums {
		total += n
	}
	return total
}

func capitalizeCategory(c Category) string {
	s := string(c)
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func formatDaysUntil(days int) string {
	switch days {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	default:
		return fmt.Sprintf("in %d days", days)
	}
}

func formatMerchants(merchants []MerchantSpend) string {
	var result string
	for _, m := range merchants {
		result += fmt.Sprintf("- %s: $%.2f (%d transactions)\n", m.Name, m.Amount, m.Count)
	}
	return result
}
