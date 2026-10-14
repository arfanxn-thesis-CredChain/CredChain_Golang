package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"CredChain_Golang/config"
)

type EmailMessage struct {
	From      string
	To        string
	Subject   string
	HTMLBody  string
	TextBody  string
	MessageID string
}

type Sender interface {
	Send(ctx context.Context, msg EmailMessage) error
}

type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	timeout  time.Duration
}

func NewSMTPSender(host string, port int, username, password string, timeout time.Duration) *SMTPSender {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		timeout:  timeout,
	}
}

func NewSMTPSenderFromConfig(cfg *config.Config) *SMTPSender {
	host := ""
	if cfg.MailHost != nil {
		host = *cfg.MailHost
	}
	port := 587
	if cfg.MailPort != nil {
		port = *cfg.MailPort
	}
	user := ""
	if cfg.MailUsername != nil {
		user = *cfg.MailUsername
	}
	pass := ""
	if cfg.MailPassword != nil {
		pass = *cfg.MailPassword
	}
	return NewSMTPSender(host, port, user, pass, 20*time.Second)
}

func hasCRLF(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

func (s *SMTPSender) Send(ctx context.Context, msg EmailMessage) error {
	if hasCRLF(msg.From) || hasCRLF(msg.To) || hasCRLF(msg.Subject) {
		return fmt.Errorf("crlf injection detected in email headers")
	}

	addr := net.JoinHostPort(s.host, strconv.Itoa(s.port))

	dialer := &net.Dialer{
		Timeout: s.timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to smtp server %s: %w", addr, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(s.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)

	closeChan := make(chan struct{})
	defer close(closeChan)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-closeChan:
		}
	}()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("failed to create smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("smtp hello failed: %w", err)
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName: s.host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls failed: %w", err)
		}
	}

	if s.username != "" {
		auth := smtp.PlainAuth("", s.username, s.password, s.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth failed: %w", err)
		}
	}

	if err := client.Mail(msg.From); err != nil {
		return fmt.Errorf("smtp MAIL FROM failed: %w", err)
	}

	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp RCPT TO failed: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA command failed: %w", err)
	}

	rawMessage := s.buildMIMEMessage(msg)
	if _, err := w.Write(rawMessage); err != nil {
		_ = w.Close()
		return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to complete smtp data write: %w", err)
	}

	return client.Quit()
}

func (s *SMTPSender) buildMIMEMessage(msg EmailMessage) []byte {
	boundary := fmt.Sprintf("=_boundary_%d", time.Now().UnixNano())
	encodedSubject := mime.QEncoding.Encode("utf-8", msg.Subject)

	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", msg.From)
	fmt.Fprintf(&b, "To: %s\r\n", msg.To)
	fmt.Fprintf(&b, "Subject: %s\r\n", encodedSubject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")

	if msg.MessageID != "" {
		msgID := strings.Trim(msg.MessageID, "<>")
		fmt.Fprintf(&b, "Message-ID: <%s>\r\n", msgID)
	}

	if msg.HTMLBody != "" && msg.TextBody != "" {
		fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)

		// Text part
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.TextBody)
		b.WriteString("\r\n\r\n")

		// HTML part
		fmt.Fprintf(&b, "--%s\r\n", boundary)
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.HTMLBody)
		b.WriteString("\r\n\r\n")

		fmt.Fprintf(&b, "--%s--\r\n", boundary)
	} else if msg.HTMLBody != "" {
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.HTMLBody)
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: 8bit\r\n\r\n")
		b.WriteString(msg.TextBody)
	}

	return []byte(b.String())
}
