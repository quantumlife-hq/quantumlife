package email

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/quantumlife/quantumlife/internal/briefing"
)

func TestDefaultConfig(t *testing.T) {
	// Clear environment
	origHost := os.Getenv("SMTP_HOST")
	origPort := os.Getenv("SMTP_PORT")
	defer func() {
		os.Setenv("SMTP_HOST", origHost)
		os.Setenv("SMTP_PORT", origPort)
	}()

	os.Setenv("SMTP_HOST", "smtp.test.com")
	os.Setenv("SMTP_PORT", "465")

	cfg := DefaultConfig()

	if cfg.SMTPHost != "smtp.test.com" {
		t.Errorf("SMTPHost = %v, want smtp.test.com", cfg.SMTPHost)
	}
	if cfg.SMTPPort != 465 {
		t.Errorf("SMTPPort = %d, want 465", cfg.SMTPPort)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestDefaultConfig_DefaultPort(t *testing.T) {
	origPort := os.Getenv("SMTP_PORT")
	defer os.Setenv("SMTP_PORT", origPort)

	os.Unsetenv("SMTP_PORT")
	cfg := DefaultConfig()

	if cfg.SMTPPort != 587 {
		t.Errorf("SMTPPort = %d, want 587 (default)", cfg.SMTPPort)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		defaultVal string
		envVal     string
		want       string
	}{
		{"uses env value", "TEST_VAR", "default", "custom", "custom"},
		{"uses default when empty", "TEST_VAR_EMPTY", "default", "", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv(tt.key, tt.envVal)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvOrDefault(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getEnvOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewSender(t *testing.T) {
	cfg := Config{
		SMTPHost:  "smtp.test.com",
		SMTPPort:  587,
		FromEmail: "test@test.com",
	}

	sender := NewSender(cfg)

	if sender == nil {
		t.Fatal("NewSender returned nil")
	}
	if sender.config.SMTPHost != cfg.SMTPHost {
		t.Error("config not set correctly")
	}
	if sender.templates == nil {
		t.Error("templates map is nil")
	}
}

func TestSender_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name: "configured",
			config: Config{
				SMTPHost:  "smtp.test.com",
				FromEmail: "test@test.com",
			},
			want: true,
		},
		{
			name: "missing host",
			config: Config{
				SMTPHost:  "",
				FromEmail: "test@test.com",
			},
			want: false,
		},
		{
			name: "missing from email",
			config: Config{
				SMTPHost:  "smtp.test.com",
				FromEmail: "",
			},
			want: false,
		},
		{
			name:   "empty config",
			config: Config{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NewSender(tt.config)
			if got := sender.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSender_Send_NotConfigured(t *testing.T) {
	sender := NewSender(Config{})
	msg := &Message{To: []string{"test@test.com"}}

	err := sender.Send(context.Background(), msg)

	if err == nil {
		t.Error("expected error for unconfigured sender")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSender_TestConnection_NotConfigured(t *testing.T) {
	sender := NewSender(Config{})

	err := sender.TestConnection(context.Background())

	if err == nil {
		t.Error("expected error for unconfigured sender")
	}
}

func TestSender_buildEmail_TextOnly(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		TextBody: "Hello World",
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "From: Test Sender <from@test.com>") {
		t.Error("should contain From header")
	}
	if !strings.Contains(emailStr, "To: to@test.com") {
		t.Error("should contain To header")
	}
	if !strings.Contains(emailStr, "Subject: Test Subject") {
		t.Error("should contain Subject header")
	}
	if !strings.Contains(emailStr, "Content-Type: text/plain") {
		t.Error("should have text/plain content type")
	}
	if !strings.Contains(emailStr, "Hello World") {
		t.Error("should contain body")
	}
}

func TestSender_buildEmail_HTMLOnly(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		HTMLBody: "<html><body>Hello</body></html>",
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "Content-Type: text/html") {
		t.Error("should have text/html content type")
	}
}

func TestSender_buildEmail_Multipart(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		TextBody: "Plain text",
		HTMLBody: "<html><body>HTML</body></html>",
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "multipart/alternative") {
		t.Error("should be multipart/alternative")
	}
	if !strings.Contains(emailStr, "Plain text") {
		t.Error("should contain text body")
	}
	if !strings.Contains(emailStr, "<html>") {
		t.Error("should contain HTML body")
	}
}

func TestSender_buildEmail_WithCC(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		CC:       []string{"cc@test.com"},
		Subject:  "Test Subject",
		TextBody: "Test",
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "Cc: cc@test.com") {
		t.Error("should contain Cc header")
	}
}

func TestSender_buildEmail_WithHeaders(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		TextBody: "Test",
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "X-Custom-Header: custom-value") {
		t.Error("should contain custom header")
	}
}

func TestSender_buildEmail_WithAttachment(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		TextBody: "See attached",
		Attachments: []Attachment{
			{
				Filename:    "test.txt",
				ContentType: "text/plain",
				Data:        []byte("file content"),
			},
		},
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "multipart/mixed") {
		t.Error("should be multipart/mixed")
	}
	if !strings.Contains(emailStr, "Content-Disposition: attachment") {
		t.Error("should have attachment disposition")
	}
	if !strings.Contains(emailStr, "test.txt") {
		t.Error("should contain filename")
	}
}

func TestSender_buildEmail_WithMultipleAttachments(t *testing.T) {
	sender := NewSender(Config{
		FromEmail: "from@test.com",
		FromName:  "Test Sender",
	})

	msg := &Message{
		To:       []string{"to@test.com"},
		Subject:  "Test Subject",
		HTMLBody: "<html>See attached</html>",
		Attachments: []Attachment{
			{Filename: "file1.txt", ContentType: "text/plain", Data: []byte("content1")},
			{Filename: "file2.pdf", ContentType: "application/pdf", Data: []byte("content2")},
		},
	}

	email, err := sender.buildEmail(msg)
	if err != nil {
		t.Fatalf("buildEmail failed: %v", err)
	}

	emailStr := string(email)
	if !strings.Contains(emailStr, "file1.txt") {
		t.Error("should contain first attachment")
	}
	if !strings.Contains(emailStr, "file2.pdf") {
		t.Error("should contain second attachment")
	}
}

func TestSender_SendBriefing(t *testing.T) {
	sender := NewSender(Config{}) // Not configured

	b := &briefing.Briefing{
		Date:    time.Now(),
		Summary: "Test briefing",
	}

	err := sender.SendBriefing(context.Background(), "test@test.com", b)

	if err == nil {
		t.Error("expected error for unconfigured sender")
	}
}

func TestSender_SendNotification(t *testing.T) {
	sender := NewSender(Config{}) // Not configured

	err := sender.SendNotification(context.Background(), "test@test.com", "Subject", "Body")

	if err == nil {
		t.Error("expected error for unconfigured sender")
	}
}

// BulkSender tests

func TestNewBulkSender(t *testing.T) {
	sender := NewSender(Config{})
	bs := NewBulkSender(sender, 10, 100*time.Millisecond)

	if bs == nil {
		t.Fatal("NewBulkSender returned nil")
	}
	if bs.batchSize != 10 {
		t.Errorf("batchSize = %d, want 10", bs.batchSize)
	}
	if bs.delay != 100*time.Millisecond {
		t.Errorf("delay = %v, want 100ms", bs.delay)
	}
}

func TestBulkSender_SendToMany_NotConfigured(t *testing.T) {
	sender := NewSender(Config{})
	bs := NewBulkSender(sender, 10, 0)

	recipients := []string{"a@test.com", "b@test.com"}
	results := bs.SendToMany(context.Background(), recipients, "Subject", "Text", "")

	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}

	// All should fail (sender not configured)
	for _, r := range results {
		if r.Success {
			t.Error("should fail for unconfigured sender")
		}
	}
}

func TestBulkSender_SendToMany_ContextCancelled(t *testing.T) {
	sender := NewSender(Config{
		SMTPHost:  "smtp.test.com",
		FromEmail: "test@test.com",
	})
	bs := NewBulkSender(sender, 10, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	recipients := []string{"a@test.com", "b@test.com"}
	results := bs.SendToMany(ctx, recipients, "Subject", "Text", "")

	// All should fail with context error
	for _, r := range results {
		if r.Success {
			t.Error("should fail when context cancelled")
		}
		if r.Error == nil {
			t.Error("should have context error")
		}
	}
}

// TemplatedSender tests

func TestNewTemplatedSender(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	if ts == nil {
		t.Fatal("NewTemplatedSender returned nil")
	}
	if ts.templates == nil {
		t.Error("templates map is nil")
	}
}

func TestTemplatedSender_RegisterTemplate(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	err := ts.RegisterTemplate("welcome", "Welcome!", "Hello {{.Name}}", "<h1>Hello {{.Name}}</h1>")

	if err != nil {
		t.Fatalf("RegisterTemplate failed: %v", err)
	}
	if _, ok := ts.templates["welcome"]; !ok {
		t.Error("template not registered")
	}
}

func TestTemplatedSender_RegisterTemplate_InvalidText(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	err := ts.RegisterTemplate("bad", "Subject", "{{.Invalid}", "")

	if err == nil {
		t.Error("expected error for invalid text template")
	}
}

func TestTemplatedSender_RegisterTemplate_InvalidHTML(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	err := ts.RegisterTemplate("bad", "Subject", "", "{{.Invalid}")

	if err == nil {
		t.Error("expected error for invalid HTML template")
	}
}

func TestTemplatedSender_SendTemplate_NotFound(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	err := ts.SendTemplate(context.Background(), "test@test.com", "nonexistent", nil)

	if err == nil {
		t.Error("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "template not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTemplatedSender_SendTemplate_TextError(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	// Register template with field that will cause error
	ts.RegisterTemplate("test", "Subject", "{{.Missing.Field}}", "")

	err := ts.SendTemplate(context.Background(), "test@test.com", "test", struct{}{})

	if err == nil {
		t.Error("expected error for template execution")
	}
}

func TestTemplatedSender_SendTemplate_HTMLError(t *testing.T) {
	sender := NewSender(Config{})
	ts := NewTemplatedSender(sender)

	// Register template with field that will cause error
	ts.RegisterTemplate("test", "Subject", "", "{{.Missing.Field}}")

	err := ts.SendTemplate(context.Background(), "test@test.com", "test", struct{}{})

	if err == nil {
		t.Error("expected error for HTML template execution")
	}
}

// MIMEWriter tests

func TestNewMIMEWriter(t *testing.T) {
	mw := NewMIMEWriter()

	if mw == nil {
		t.Fatal("NewMIMEWriter returned nil")
	}
	if mw.boundary == "" {
		t.Error("boundary should not be empty")
	}
	if mw.header == nil {
		t.Error("header should not be nil")
	}
}

func TestMIMEWriter_Boundary(t *testing.T) {
	mw := NewMIMEWriter()

	boundary := mw.Boundary()
	if boundary == "" {
		t.Error("Boundary should not be empty")
	}
	if !strings.Contains(boundary, "MIME") {
		t.Error("Boundary should contain 'MIME'")
	}
}

func TestMIMEWriter_AddPart(t *testing.T) {
	mw := NewMIMEWriter()

	err := mw.AddPart("text/plain", []byte("Hello"))
	if err != nil {
		t.Fatalf("AddPart failed: %v", err)
	}

	data := mw.Bytes()
	if len(data) == 0 {
		t.Error("Bytes should not be empty after AddPart")
	}
}

func TestMIMEWriter_Bytes_Empty(t *testing.T) {
	mw := NewMIMEWriter()

	data := mw.Bytes()
	if len(data) != 0 {
		t.Error("Bytes should be empty initially")
	}
}

// Struct field tests

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		SMTPHost:    "smtp.test.com",
		SMTPPort:    587,
		Username:    "user",
		Password:    "pass",
		FromEmail:   "from@test.com",
		FromName:    "Sender",
		UseTLS:      true,
		UseStartTLS: false,
		Timeout:     60 * time.Second,
	}

	if cfg.SMTPHost != "smtp.test.com" {
		t.Error("SMTPHost not set correctly")
	}
	if cfg.SMTPPort != 587 {
		t.Error("SMTPPort not set correctly")
	}
	if cfg.Username != "user" {
		t.Error("Username not set correctly")
	}
	if cfg.Timeout != 60*time.Second {
		t.Error("Timeout not set correctly")
	}
}

func TestMessage_Fields(t *testing.T) {
	msg := Message{
		To:          []string{"to@test.com"},
		CC:          []string{"cc@test.com"},
		BCC:         []string{"bcc@test.com"},
		Subject:     "Subject",
		TextBody:    "Text",
		HTMLBody:    "<html>HTML</html>",
		Attachments: []Attachment{},
		Headers:     map[string]string{"X-Test": "value"},
	}

	if len(msg.To) != 1 {
		t.Error("To not set correctly")
	}
	if len(msg.CC) != 1 {
		t.Error("CC not set correctly")
	}
	if len(msg.BCC) != 1 {
		t.Error("BCC not set correctly")
	}
	if msg.Subject != "Subject" {
		t.Error("Subject not set correctly")
	}
}

func TestAttachment_Fields(t *testing.T) {
	att := Attachment{
		Filename:    "file.txt",
		ContentType: "text/plain",
		Data:        []byte("content"),
	}

	if att.Filename != "file.txt" {
		t.Error("Filename not set correctly")
	}
	if att.ContentType != "text/plain" {
		t.Error("ContentType not set correctly")
	}
	if !bytes.Equal(att.Data, []byte("content")) {
		t.Error("Data not set correctly")
	}
}

func TestSendResult_Fields(t *testing.T) {
	result := SendResult{
		Recipient: "test@test.com",
		Success:   true,
		Error:     nil,
	}

	if result.Recipient != "test@test.com" {
		t.Error("Recipient not set correctly")
	}
	if !result.Success {
		t.Error("Success not set correctly")
	}
}

func TestEmailTemplate_Fields(t *testing.T) {
	tmpl := EmailTemplate{
		Name:     "welcome",
		Subject:  "Welcome!",
		TextTmpl: nil,
		HTMLTmpl: nil,
	}

	if tmpl.Name != "welcome" {
		t.Error("Name not set correctly")
	}
	if tmpl.Subject != "Welcome!" {
		t.Error("Subject not set correctly")
	}
}
