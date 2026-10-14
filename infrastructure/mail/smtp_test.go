package mail

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSMTPSender_HeaderInjection_ReturnsError(t *testing.T) {
	sender := NewSMTPSender("localhost", 2525, "", "", 5*time.Second)

	tests := []struct {
		name string
		msg  EmailMessage
	}{
		{
			name: "CR in subject",
			msg:  EmailMessage{From: "a@b.com", To: "c@d.com", Subject: "bad\rsubject"},
		},
		{
			name: "LF in subject",
			msg:  EmailMessage{From: "a@b.com", To: "c@d.com", Subject: "bad\nsubject"},
		},
		{
			name: "CRLF in recipient",
			msg:  EmailMessage{From: "a@b.com", To: "c@d.com\r\nBcc: evil@e.com", Subject: "subj"},
		},
		{
			name: "LF in sender",
			msg:  EmailMessage{From: "a@b.com\n", To: "c@d.com", Subject: "subj"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sender.Send(context.Background(), tt.msg)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "crlf")
		})
	}
}

func TestSMTPSender_ConnectionRefused_ReturnsError(t *testing.T) {
	// Use an unused port
	sender := NewSMTPSender("127.0.0.1", 54321, "", "", 1*time.Second)
	err := sender.Send(context.Background(), EmailMessage{
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Test",
		TextBody: "Hello",
	})
	assert.Error(t, err)
}

func TestSMTPSender_ContextCancelled_ReturnsError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	sender := NewSMTPSender("127.0.0.1", port, "", "", 5*time.Second)
	sendErr := sender.Send(ctx, EmailMessage{
		From:     "sender@example.com",
		To:       "recipient@example.com",
		Subject:  "Test",
		TextBody: "Hello",
	})
	assert.Error(t, sendErr)
}

func TestSMTPSender_SendSuccess_MockSMTPServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port

	var receivedData strings.Builder
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		// 220 banner
		conn.Write([]byte("220 mock.smtp ready\r\n"))

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")

			if strings.HasPrefix(line, "EHLO") || strings.HasPrefix(line, "HELO") {
				conn.Write([]byte("250-mock.smtp\r\n250 HELP\r\n"))
			} else if strings.HasPrefix(line, "MAIL FROM:") {
				conn.Write([]byte("250 OK\r\n"))
			} else if strings.HasPrefix(line, "RCPT TO:") {
				conn.Write([]byte("250 OK\r\n"))
			} else if line == "DATA" {
				conn.Write([]byte("354 Start mail input; end with <CRLF>.<CRLF>\r\n"))
				for {
					dataLine, err := reader.ReadString('\n')
					if err != nil {
						return
					}
					if dataLine == ".\r\n" {
						break
					}
					receivedData.WriteString(dataLine)
				}
				conn.Write([]byte("250 OK: message queued\r\n"))
			} else if line == "QUIT" {
				conn.Write([]byte("221 Bye\r\n"))
				return
			}
		}
	}()

	sender := NewSMTPSender("127.0.0.1", port, "", "", 5*time.Second)
	msg := EmailMessage{
		From:      "sender@example.com",
		To:        "recipient@example.com",
		Subject:   "Submission Approved",
		HTMLBody:  "<p>Your submission was approved.</p>",
		TextBody:  "Your submission was approved.",
		MessageID: "test-msg-123",
	}

	err = sender.Send(context.Background(), msg)
	assert.NoError(t, err)

	<-done
	body := receivedData.String()
	assert.Contains(t, body, "Subject: Submission Approved")
	assert.Contains(t, body, "From: sender@example.com")
	assert.Contains(t, body, "To: recipient@example.com")
	assert.Contains(t, body, "Message-ID: <test-msg-123>")
	assert.Contains(t, body, "Your submission was approved.")
}

func generateSelfSignedCert(t *testing.T) tls.Certificate {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Mock SMTP"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err)
	return cert
}
