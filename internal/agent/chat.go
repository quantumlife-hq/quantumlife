// Package agent implements the QuantumLife agent.
package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/quantumlife/quantumlife/internal/llm"
)

// ChatSession manages an interactive chat session
type ChatSession struct {
	agent   *Agent
	history []llm.Message
}

// NewChatSession creates a new chat session
func NewChatSession(agent *Agent) *ChatSession {
	return &ChatSession{
		agent:   agent,
		history: make([]llm.Message, 0),
	}
}

// SendMessage sends a message and gets a response
func (s *ChatSession) SendMessage(ctx context.Context, message string) (string, error) {
	response, err := s.agent.Chat(ctx, message, s.history)
	if err != nil {
		return "", err
	}

	// Add to history
	s.history = append(s.history, llm.Message{Role: "user", Content: message})
	s.history = append(s.history, llm.Message{Role: "assistant", Content: response})

	// Keep history manageable (last 20 messages)
	if len(s.history) > 20 {
		s.history = s.history[len(s.history)-20:]
	}

	return response, nil
}

// Clear clears chat history
func (s *ChatSession) Clear() {
	s.history = make([]llm.Message, 0)
}

// RunInteractive runs an interactive chat in the terminal
func (s *ChatSession) RunInteractive(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("QuantumLife Agent")
	fmt.Println("   Type 'exit' to quit, 'clear' to reset conversation")
	fmt.Println()

	for {
		fmt.Print("You: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit", "bye":
			fmt.Println("\nGoodbye!")
			return nil
		case "clear":
			s.Clear()
			fmt.Println("Conversation cleared")
			fmt.Println()
			continue
		case "stats":
			stats, _ := s.agent.GetStats(ctx)
			fmt.Printf("\nItems: %d | Memories: %d | Running: %v\n\n",
				stats.TotalItems, stats.TotalMemories, stats.Running)
			continue
		}

		response, err := s.SendMessage(ctx, input)
		if err != nil {
			fmt.Printf("\nError: %v\n\n", err)
			continue
		}

		fmt.Printf("\nAgent: %s\n\n", response)
	}
}
