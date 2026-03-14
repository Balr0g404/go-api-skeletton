package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPSender sends emails via SMTP (supports STARTTLS on 587, TLS on 465).
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSMTPSender creates an SMTPSender from Config.
func NewSMTPSender(cfg Config) *SMTPSender {
	return &SMTPSender{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.From,
	}
}

func (s *SMTPSender) Send(msg Message) error {
	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	body := s.buildMIME(msg)

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	if s.port == 465 {
		return s.sendTLS(addr, auth, body, msg.To)
	}
	// Port 587 or 25: smtp.SendMail handles STARTTLS negotiation.
	return smtp.SendMail(addr, auth, s.from, []string{msg.To}, body)
}

func (s *SMTPSender) sendTLS(addr string, auth smtp.Auth, body []byte, to string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("smtp: TLS dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp: new client: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp: auth: %w", err)
		}
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp: DATA: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("smtp: write: %w", err)
	}
	return w.Close()
}

func (s *SMTPSender) buildMIME(msg Message) []byte {
	var b strings.Builder
	b.WriteString("From: " + s.from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg.HTML)
	return []byte(b.String())
}
