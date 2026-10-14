package mail

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"CredChain_Golang/config"
)

type mockSender struct {
	mock.Mock
}

func (m *mockSender) Send(ctx context.Context, msg EmailMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func TestDefaultMailer_RenderHTML_SingleLanguageAndButtons(t *testing.T) {
	mailer := NewMailer(&config.Config{}, nil)
	data := EmailData{
		Language: "id", Title: "Pengajuan Disetujui", Greeting: "Halo John",
		Message: "Pengajuan kredensial Anda telah disetujui.",
		Details: []EmailDetail{{Key: "Nomor Kredensial", Value: "CRD-12345"}},
		Buttons: []EmailButton{
			{Label: "Lihat Kredensial", URL: "http://localhost:5173/credentials/CRD-12345"},
			{Label: "Aksi Kedua", URL: "http://localhost:5173/secondary"},
			{Label: "Aksi Ketiga", URL: "http://localhost:5173/tertiary"},
		},
		FallbackURL:  "http://localhost:5173/credentials/CRD-12345",
		FallbackText: "Jika tombol tidak berfungsi, gunakan tautan berikut:",
		Footer:       "Ini adalah notifikasi otomatis. Jangan balas email ini.",
	}
	html, err := mailer.RenderHTML(data)
	require.NoError(t, err)
	for _, expected := range []string{`<html lang="id">`, "Halo John", data.Message, "Nomor Kredensial", "Aksi Ketiga", data.FallbackText, data.Footer, "#EAB308", "#0F172A", "#E2E8F0"} {
		assert.Contains(t, html, expected)
	}
	for _, unexpected := range []string{"Hello John", "View Credential", "All rights reserved", "If the buttons above"} {
		assert.NotContains(t, html, unexpected)
	}
}

func TestDefaultMailer_RenderHTML_NarrowLayoutWithLongNumber(t *testing.T) {
	mailer := NewMailer(&config.Config{}, nil)
	longNumber := "CERTIFICATE-2026-" + strings.Repeat("1234567890", 15)
	html, err := mailer.RenderHTML(EmailData{
		Language: "en", Details: []EmailDetail{{Key: "Credential Number", Value: longNumber}},
	})
	require.NoError(t, err)
	assert.Contains(t, html, `class="email-container" width="100%"`)
	assert.Contains(t, html, `width:100%; max-width:600px`)
	assert.Equal(t, 1, strings.Count(html, longNumber))
	assert.Equal(t, 1, strings.Count(html, `word-break: break-all`))
	assert.Contains(t, html, "Credential Number")
	assert.NotContains(t, html, "Nomor Kredensial")
}

func TestDefaultMailer_RenderText_SingleLanguage(t *testing.T) {
	mailer := NewMailer(&config.Config{}, nil)
	data := EmailData{
		Language: "en", Title: "Submission Approved", Greeting: "Hello John",
		Message:      "Your credential submission was approved.",
		Details:      []EmailDetail{{Key: "Credential Number", Value: "CRD-12345"}},
		Buttons:      []EmailButton{{Label: "View", URL: "http://localhost:5173/credentials/CRD-12345"}},
		FallbackURL:  "http://localhost:5173/credentials/CRD-12345",
		FallbackText: "Use this link:", Footer: "Automated notification. Do not reply.",
	}
	text := mailer.RenderText(data)
	for _, expected := range []string{"Hello John", data.Message, "Credential Number: CRD-12345", "[View]: http://localhost:5173/credentials/CRD-12345", data.FallbackText, data.Footer} {
		assert.Contains(t, text, expected)
	}
	assert.NotContains(t, text, "Halo John")
	assert.NotContains(t, text, "All rights reserved")
}

func TestDefaultMailer_SendEmail_Disabled(t *testing.T) {
	cfg := &config.Config{MailHost: nil}
	sender := new(mockSender)
	mailer := NewMailer(cfg, sender)
	err := mailer.SendEmail(context.Background(), "user@example.com", EmailData{Title: "Test"})
	assert.NoError(t, err)
	sender.AssertNotCalled(t, "Send")
}

func TestDefaultMailer_SendEmail_InvalidRecipient(t *testing.T) {
	host := "smtp.example.com"
	cfg := &config.Config{MailHost: &host}
	sender := new(mockSender)
	mailer := NewMailer(cfg, sender)
	err := mailer.SendEmail(context.Background(), "invalid-email", EmailData{Title: "Test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient")
	sender.AssertNotCalled(t, "Send")
}

func TestDefaultMailer_SendEmail_Success(t *testing.T) {
	host := "smtp.example.com"
	from := "noreply@credchain.com"
	cfg := &config.Config{MailHost: &host, MailFrom: &from}
	sender := new(mockSender)
	mailer := NewMailer(cfg, sender)
	sender.On("Send", mock.Anything, mock.MatchedBy(func(msg EmailMessage) bool {
		return msg.From == from && msg.To == "recipient@example.com" &&
			msg.Subject == "Submission Approved" &&
			strings.Contains(msg.HTMLBody, "Hello") &&
			strings.Contains(msg.TextBody, "Hello")
	})).Return(nil)
	err := mailer.SendEmail(context.Background(), "recipient@example.com", EmailData{
		Language: "en", Title: "Submission Approved", Message: "Hello",
	})
	assert.NoError(t, err)
	sender.AssertExpectations(t)
}
