package mail

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"net/mail"
	"strings"
	"time"

	"CredChain_Golang/config"
)

//go:embed template.html
var emailTemplateHTML string

var emailTemplate = template.Must(template.New("email").Parse(emailTemplateHTML))

type EmailButton struct {
	Label string
	URL   string
}

type EmailDetail struct {
	Key   string
	Value string
}

type EmailData struct {
	Language     string
	Title        string
	Greeting     string
	Message      string
	Details      []EmailDetail
	Buttons      []EmailButton
	FallbackURL  string
	FallbackText string
	Footer       string
}

type Mailer interface {
	SendEmail(ctx context.Context, to string, data EmailData) error
}

type DefaultMailer struct {
	sender Sender
	cfg    *config.Config
}

func NewMailer(cfg *config.Config, sender Sender) *DefaultMailer {
	return &DefaultMailer{
		sender: sender,
		cfg:    cfg,
	}
}

func ProvideMailer(cfg *config.Config) Mailer {
	return NewMailer(cfg, NewSMTPSenderFromConfig(cfg))
}

type templateViewData struct {
	EmailData
	Year int
}

func (m *DefaultMailer) RenderHTML(data EmailData) (string, error) {
	view := templateViewData{
		EmailData: data,
		Year:      time.Now().Year(),
	}
	var buf bytes.Buffer
	if err := emailTemplate.Execute(&buf, view); err != nil {
		return "", fmt.Errorf("failed to execute email template: %w", err)
	}
	return buf.String(), nil
}

func (m *DefaultMailer) RenderText(data EmailData) string {
	var b strings.Builder
	if data.Greeting != "" {
		b.WriteString(data.Greeting + ",\n\n")
	}
	if data.Message != "" {
		b.WriteString(data.Message + "\n\n")
	}
	for _, d := range data.Details {
		fmt.Fprintf(&b, "%s: %s\n", d.Key, d.Value)
	}
	if len(data.Details) > 0 {
		b.WriteString("\n")
	}
	for _, btn := range data.Buttons {
		fmt.Fprintf(&b, "[%s]: %s\n", btn.Label, btn.URL)
	}
	if len(data.Buttons) > 0 {
		b.WriteString("\n")
	}
	if data.FallbackURL != "" {
		fmt.Fprintf(&b, "%s\n%s\n\n", data.FallbackText, data.FallbackURL)
	}
	if data.Footer != "" {
		fmt.Fprintf(&b, "© %d CredChain. %s\n", time.Now().Year(), data.Footer)
	}
	return b.String()
}

func (m *DefaultMailer) SendEmail(ctx context.Context, to string, data EmailData) error {
	if !m.cfg.IsMailEnabled() {
		return nil
	}

	if _, err := mail.ParseAddress(to); err != nil {
		return fmt.Errorf("invalid recipient email address: %w", err)
	}

	htmlContent, err := m.RenderHTML(data)
	if err != nil {
		return err
	}

	textContent := m.RenderText(data)

	subject := data.Title

	from := "no-reply@credchain.com"
	if m.cfg.MailFrom != nil && *m.cfg.MailFrom != "" {
		from = *m.cfg.MailFrom
	}

	msg := EmailMessage{
		From:      from,
		To:        to,
		Subject:   subject,
		HTMLBody:  htmlContent,
		TextBody:  textContent,
		MessageID: fmt.Sprintf("%d-%s", time.Now().UnixNano(), to),
	}

	return m.sender.Send(ctx, msg)
}
