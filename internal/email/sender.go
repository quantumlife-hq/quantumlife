// Package email provides email sending capabilities.
package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html/template"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"

	"github.com/quantumlife/quantumlife/internal/briefing"
)

// Sender handles email delivery
type Sender struct {
	config     Config
	templates  map[string]*template.Template
}

// Config configures the email sender
type Config struct {
	SMTPHost     string
	SMTPPort     int
	Username     string
	Password     string
	FromEmail    string
	FromName     string
	UseTLS       bool
	UseStartTLS  bool
	Timeout      time.Duration
}

// DefaultConfig returns config from environment
func DefaultConfig() Config {
	port := 587
	if os.Getenv("SMTP_PORT") != "" {
		fmt.Sscanf(os.Getenv("SMTP_PORT"), "%d", &port)
	}

	return Config{
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    port,
		Username:    os.Getenv("SMTP_USERNAME"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		FromEmail:   os.Getenv("SMTP_FROM_EMAIL"),
		FromName:    getEnvOrDefault("SMTP_FROM_NAME", "QuantumLife"),
		UseTLS:      os.Getenv("SMTP_USE_TLS") == "true",
		UseStartTLS: getEnvOrDefault("SMTP_USE_STARTTLS", "true") == "true",
		Timeout:     30 * time.Second,
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// NewSender creates a new email sender
func NewSender(cfg Config) *Sender {
	return &Sender{
		config:    cfg,
		templates: make(map[string]*template.Template),
	}
}

// Message represents an email message
type Message struct {
	To          []string
	CC          []string
	BCC         []string
	Subject     string
	TextBody    string
	HTMLBody    string
	Attachments []Attachment
	Headers     map[string]string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
}

// Send sends an email message
func (s *Sender) Send(ctx context.Context, msg *Message) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email sender not configured")
	}

	// Build email
	email, err := s.buildEmail(msg)
	if err != nil {
		return fmt.Errorf("failed to build email: %w", err)
	}

	// Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)

	var conn net.Conn
	dialer := net.Dialer{Timeout: s.config.Timeout}

	if s.config.UseTLS {
		// Direct TLS connection
		tlsConfig := &tls.Config{ServerName: s.config.SMTPHost}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// STARTTLS if needed
	if s.config.UseStartTLS && !s.config.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			tlsConfig := &tls.Config{ServerName: s.config.SMTPHost}
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("STARTTLS failed: %w", err)
			}
		}
	}

	// Authenticate
	if s.config.Username != "" && s.config.Password != "" {
		auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(s.config.FromEmail); err != nil {
		return fmt.Errorf("MAIL FROM failed: %w", err)
	}

	// Set recipients
	allRecipients := append(append(msg.To, msg.CC...), msg.BCC...)
	for _, rcpt := range allRecipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO failed for %s: %w", rcpt, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	if _, err := w.Write(email); err != nil {
		return fmt.Errorf("failed to write email data: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}

// buildEmail constructs the raw email bytes
func (s *Sender) buildEmail(msg *Message) ([]byte, error) {
	var buf bytes.Buffer

	// Generate boundary for multipart
	boundary := fmt.Sprintf("----=_Part_%d", time.Now().UnixNano())

	// Headers
	buf.WriteString(fmt.Sprintf("From: %s <%s>\r\n", s.config.FromName, s.config.FromEmail))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	if len(msg.CC) > 0 {
		buf.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(msg.CC, ", ")))
	}
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	buf.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for key, value := range msg.Headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// Determine content type
	hasHTML := msg.HTMLBody != ""
	hasText := msg.TextBody != ""
	hasAttachments := len(msg.Attachments) > 0

	if hasAttachments {
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		// Text/HTML part
		if hasHTML || hasText {
			altBoundary := fmt.Sprintf("----=_Alt_%d", time.Now().UnixNano())
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", altBoundary))
			buf.WriteString("\r\n")

			if hasText {
				buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
				buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
				buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
				buf.WriteString(msg.TextBody)
				buf.WriteString("\r\n")
			}

			if hasHTML {
				buf.WriteString(fmt.Sprintf("--%s\r\n", altBoundary))
				buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
				buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
				buf.WriteString(msg.HTMLBody)
				buf.WriteString("\r\n")
			}

			buf.WriteString(fmt.Sprintf("--%s--\r\n", altBoundary))
		}

		// Attachments
		for _, att := range msg.Attachments {
			buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
			buf.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", att.ContentType, att.Filename))
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", att.Filename))
			buf.WriteString(base64.StdEncoding.EncodeToString(att.Data))
			buf.WriteString("\r\n")
		}

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasHTML && hasText {
		// Multipart alternative
		buf.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary))
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(msg.TextBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		buf.WriteString(msg.HTMLBody)
		buf.WriteString("\r\n")

		buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))
	} else if hasHTML {
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		buf.WriteString(msg.HTMLBody)
	} else {
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(msg.TextBody)
	}

	return buf.Bytes(), nil
}

// SendBriefing sends a daily briefing email
func (s *Sender) SendBriefing(ctx context.Context, to string, b *briefing.Briefing) error {
	msg := &Message{
		To:       []string{to},
		Subject:  fmt.Sprintf("Your Daily Briefing - %s", b.Date.Format("January 2, 2006")),
		TextBody: b.RenderText(),
		HTMLBody: b.RenderHTML(),
		Headers: map[string]string{
			"X-QuantumLife-Type": "daily-briefing",
			"X-QuantumLife-Date": b.Date.Format(time.RFC3339),
		},
	}

	return s.Send(ctx, msg)
}

// SendNotification sends a simple notification email
func (s *Sender) SendNotification(ctx context.Context, to, subject, body string) error {
	msg := &Message{
		To:       []string{to},
		Subject:  subject,
		TextBody: body,
		Headers: map[string]string{
			"X-QuantumLife-Type": "notification",
		},
	}

	return s.Send(ctx, msg)
}

// IsConfigured checks if the sender is properly configured
func (s *Sender) IsConfigured() bool {
	return s.config.SMTPHost != "" && s.config.FromEmail != ""
}

// TestConnection tests the SMTP connection
func (s *Sender) TestConnection(ctx context.Context) error {
	if !s.IsConfigured() {
		return fmt.Errorf("email sender not configured")
	}

	addr := fmt.Sprintf("%s:%d", s.config.SMTPHost, s.config.SMTPPort)
	dialer := net.Dialer{Timeout: s.config.Timeout}

	var conn net.Conn
	var err error

	if s.config.UseTLS {
		tlsConfig := &tls.Config{ServerName: s.config.SMTPHost}
		conn, err = tls.DialWithDialer(&dialer, "tcp", addr, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.SMTPHost)
	if err != nil {
		return fmt.Errorf("SMTP client creation failed: %w", err)
	}
	defer client.Close()

	if err := client.Hello("quantumlife"); err != nil {
		return fmt.Errorf("HELO failed: %w", err)
	}

	return client.Quit()
}

// BulkSender handles sending emails in bulk
type BulkSender struct {
	sender     *Sender
	batchSize  int
	delay      time.Duration
}

// NewBulkSender creates a bulk sender
func NewBulkSender(sender *Sender, batchSize int, delay time.Duration) *BulkSender {
	return &BulkSender{
		sender:    sender,
		batchSize: batchSize,
		delay:     delay,
	}
}

// SendResult tracks the result of a bulk send
type SendResult struct {
	Recipient string
	Success   bool
	Error     error
}

// SendToMany sends the same message to multiple recipients
func (bs *BulkSender) SendToMany(ctx context.Context, recipients []string, subject, textBody, htmlBody string) []SendResult {
	results := make([]SendResult, len(recipients))

	for i, recipient := range recipients {
		select {
		case <-ctx.Done():
			// Context cancelled, mark remaining as failed
			for j := i; j < len(recipients); j++ {
				results[j] = SendResult{
					Recipient: recipients[j],
					Success:   false,
					Error:     ctx.Err(),
				}
			}
			return results
		default:
		}

		msg := &Message{
			To:       []string{recipient},
			Subject:  subject,
			TextBody: textBody,
			HTMLBody: htmlBody,
		}

		err := bs.sender.Send(ctx, msg)
		results[i] = SendResult{
			Recipient: recipient,
			Success:   err == nil,
			Error:     err,
		}

		// Rate limiting
		if i > 0 && i%bs.batchSize == 0 && bs.delay > 0 {
			time.Sleep(bs.delay)
		}
	}

	return results
}

// TemplatedSender sends templated emails
type TemplatedSender struct {
	sender    *Sender
	templates map[string]*EmailTemplate
}

// EmailTemplate represents an email template
type EmailTemplate struct {
	Name     string
	Subject  string
	TextTmpl *template.Template
	HTMLTmpl *template.Template
}

// NewTemplatedSender creates a templated sender
func NewTemplatedSender(sender *Sender) *TemplatedSender {
	return &TemplatedSender{
		sender:    sender,
		templates: make(map[string]*EmailTemplate),
	}
}

// RegisterTemplate registers an email template
func (ts *TemplatedSender) RegisterTemplate(name, subject, textTemplate, htmlTemplate string) error {
	et := &EmailTemplate{
		Name:    name,
		Subject: subject,
	}

	if textTemplate != "" {
		tmpl, err := template.New(name + "_text").Parse(textTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse text template: %w", err)
		}
		et.TextTmpl = tmpl
	}

	if htmlTemplate != "" {
		tmpl, err := template.New(name + "_html").Parse(htmlTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse HTML template: %w", err)
		}
		et.HTMLTmpl = tmpl
	}

	ts.templates[name] = et
	return nil
}

// SendTemplate sends an email using a registered template
func (ts *TemplatedSender) SendTemplate(ctx context.Context, to, templateName string, data interface{}) error {
	tmpl, ok := ts.templates[templateName]
	if !ok {
		return fmt.Errorf("template not found: %s", templateName)
	}

	msg := &Message{
		To:      []string{to},
		Subject: tmpl.Subject,
	}

	if tmpl.TextTmpl != nil {
		var buf bytes.Buffer
		if err := tmpl.TextTmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute text template: %w", err)
		}
		msg.TextBody = buf.String()
	}

	if tmpl.HTMLTmpl != nil {
		var buf bytes.Buffer
		if err := tmpl.HTMLTmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute HTML template: %w", err)
		}
		msg.HTMLBody = buf.String()
	}

	return ts.sender.Send(ctx, msg)
}

// MIMEWriter helps construct MIME messages
type MIMEWriter struct {
	buf      bytes.Buffer
	boundary string
	header   textproto.MIMEHeader
}

// NewMIMEWriter creates a new MIME writer
func NewMIMEWriter() *MIMEWriter {
	return &MIMEWriter{
		boundary: fmt.Sprintf("----=_MIME_%d", time.Now().UnixNano()),
		header:   make(textproto.MIMEHeader),
	}
}

// AddPart adds a MIME part
func (m *MIMEWriter) AddPart(contentType string, body []byte) error {
	writer := multipart.NewWriter(&m.buf)
	if err := writer.SetBoundary(m.boundary); err != nil {
		return err
	}

	part, err := writer.CreatePart(textproto.MIMEHeader{
		"Content-Type": {contentType},
	})
	if err != nil {
		return err
	}

	_, err = part.Write(body)
	return err
}

// Bytes returns the constructed MIME message
func (m *MIMEWriter) Bytes() []byte {
	return m.buf.Bytes()
}

// Boundary returns the MIME boundary
func (m *MIMEWriter) Boundary() string {
	return m.boundary
}
