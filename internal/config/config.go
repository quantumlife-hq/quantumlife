// Package config handles QuantumLife configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds all configuration
type Config struct {
	// Paths
	DataDir string `json:"data_dir"`

	// Server
	Server ServerConfig `json:"server"`

	// Services
	Qdrant QdrantConfig `json:"qdrant"`
	Ollama OllamaConfig `json:"ollama"`
	Claude ClaudeConfig `json:"claude"`

	// Features
	Features FeatureConfig `json:"features"`
}

// ServerConfig for HTTP server
type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// QdrantConfig for vector database
type QdrantConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// OllamaConfig for local LLM
type OllamaConfig struct {
	URL   string `json:"url"`
	Model string `json:"model"`
}

// ClaudeConfig for Claude API
type ClaudeConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// FeatureConfig for feature flags
type FeatureConfig struct {
	EnableSync   bool `json:"enable_sync"`
	EnableAgent  bool `json:"enable_agent"`
	EnableMemory bool `json:"enable_memory"`
	DebugMode    bool `json:"debug_mode"`
}

// Default returns default configuration
func Default() *Config {
	home, _ := os.UserHomeDir()

	return &Config{
		DataDir: filepath.Join(home, ".quantumlife"),
		Server: ServerConfig{
			Port: 8080,
			Host: "localhost",
		},
		Qdrant: QdrantConfig{
			Host: "localhost",
			Port: 6334,
		},
		Ollama: OllamaConfig{
			URL:   "http://localhost:11434",
			Model: "nomic-embed-text",
		},
		Claude: ClaudeConfig{
			APIKey: os.Getenv("ANTHROPIC_API_KEY"),
			Model:  "claude-sonnet-4-20250514",
		},
		Features: FeatureConfig{
			EnableSync:   true,
			EnableAgent:  true,
			EnableMemory: true,
			DebugMode:    false,
		},
	}
}

// Load loads config from file, falling back to defaults
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		path = filepath.Join(cfg.DataDir, "config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Override API key from env if set
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		cfg.Claude.APIKey = apiKey
	}

	return cfg, nil
}

// Save saves config to file
func (c *Config) Save(path string) error {
	if path == "" {
		path = filepath.Join(c.DataDir, "config.json")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	// Don't save API key to file
	safeCfg := *c
	safeCfg.Claude.APIKey = ""

	data, err := json.MarshalIndent(safeCfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
