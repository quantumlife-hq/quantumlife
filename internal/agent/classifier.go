// Package agent implements the QuantumLife agent.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quantumlife/quantumlife/internal/core"
	"github.com/quantumlife/quantumlife/internal/llm"
	"github.com/quantumlife/quantumlife/internal/storage"
)

// Classifier routes items to the correct hat
type Classifier struct {
	llm      *llm.Client
	hatStore *storage.HatStore
}

// NewClassifier creates a classifier
func NewClassifier(llmClient *llm.Client, hatStore *storage.HatStore) *Classifier {
	return &Classifier{
		llm:      llmClient,
		hatStore: hatStore,
	}
}

// ClassificationResult is the output of classification
type ClassificationResult struct {
	HatID       core.HatID `json:"hat_id"`
	Confidence  float64    `json:"confidence"`
	Priority    int        `json:"priority"`
	Sentiment   string     `json:"sentiment"`
	Summary     string     `json:"summary"`
	Entities    []string   `json:"entities"`
	ActionItems []string   `json:"action_items"`
	Reasoning   string     `json:"reasoning"`
}

// ClassifyItem determines which hat an item belongs to
func (c *Classifier) ClassifyItem(ctx context.Context, item *core.Item) (*ClassificationResult, error) {
	// Get available hats
	hats, err := c.hatStore.GetActive()
	if err != nil {
		return nil, fmt.Errorf("failed to get hats: %w", err)
	}

	// Build hat descriptions
	var hatDescriptions []string
	for _, h := range hats {
		hatDescriptions = append(hatDescriptions, fmt.Sprintf(
			"- %s (%s): %s", h.ID, h.Name, h.Description,
		))
	}

	systemPrompt := fmt.Sprintf(`You are the QuantumLife classification agent. Your job is to analyze incoming items and route them to the correct life domain (hat).

Available hats:
%s

Analyze the item and respond with ONLY a JSON object (no markdown, no explanation):
{
    "hat_id": "the hat ID this belongs to",
    "confidence": 0.0-1.0 how confident you are,
    "priority": 1-5 where 1 is urgent and 5 is low priority,
    "sentiment": "positive", "negative", or "neutral",
    "summary": "one sentence summary",
    "entities": ["list", "of", "people", "places", "orgs"],
    "action_items": ["any", "actions", "required"],
    "reasoning": "brief explanation of why this hat"
}`, strings.Join(hatDescriptions, "\n"))

	userPrompt := fmt.Sprintf(`Classify this item:

Type: %s
From: %s
Subject: %s
Content:
%s`, item.Type, item.From, item.Subject, truncateContent(item.Body, 2000))

	response, err := c.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM classification failed: %w", err)
	}

	// Parse response
	var result ClassificationResult

	// Clean response (remove markdown if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse classification: %w (response: %s)", err, response)
	}

	return &result, nil
}

// QuickClassify does a fast classification without full analysis
func (c *Classifier) QuickClassify(ctx context.Context, content string) (core.HatID, float64, error) {
	hats, err := c.hatStore.GetActive()
	if err != nil {
		return "", 0, err
	}

	var hatList []string
	for _, h := range hats {
		hatList = append(hatList, fmt.Sprintf("%s:%s", h.ID, h.Name))
	}

	systemPrompt := `You are a quick classifier. Given content, respond with ONLY the hat_id and confidence as JSON: {"hat_id": "xxx", "confidence": 0.X}`
	userPrompt := fmt.Sprintf("Hats: %s\n\nContent: %s", strings.Join(hatList, ", "), truncateContent(content, 500))

	response, err := c.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", 0, err
	}

	var result struct {
		HatID      string  `json:"hat_id"`
		Confidence float64 `json:"confidence"`
	}

	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimSuffix(response, "```")

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return core.HatPersonal, 0.5, nil // Default to personal
	}

	return core.HatID(result.HatID), result.Confidence, nil
}

func truncateContent(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
