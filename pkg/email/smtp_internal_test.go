package email

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSMTPSender_buildMIME(t *testing.T) {
	s := &SMTPSender{from: "from@example.com"}
	msg := Message{To: "to@example.com", Subject: "Hello World", HTML: "<p>Hi</p>"}
	body := string(s.buildMIME(msg))

	assert.Contains(t, body, "From: from@example.com\r\n")
	assert.Contains(t, body, "To: to@example.com\r\n")
	assert.Contains(t, body, "Subject: Hello World\r\n")
	assert.Contains(t, body, "MIME-Version: 1.0\r\n")
	assert.Contains(t, body, "Content-Type: text/html; charset=UTF-8\r\n")
	assert.Contains(t, body, "<p>Hi</p>")
}

func TestSMTPSender_Send_ConnectionRefused(t *testing.T) {
	// Port 1 is never listening; smtp.SendMail will return a connection error.
	s := &SMTPSender{host: "127.0.0.1", port: 1, from: "from@example.com"}
	err := s.Send(Message{To: "to@example.com", Subject: "test", HTML: "<p>x</p>"})
	assert.Error(t, err)
}

func TestSMTPSender_Send_TLSConnectionRefused(t *testing.T) {
	// Port 465 with nothing listening → TLS dial error.
	s := &SMTPSender{host: "127.0.0.1", port: 465, from: "from@example.com"}
	err := s.Send(Message{To: "to@example.com", Subject: "test", HTML: "<p>x</p>"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp: TLS dial")
}
