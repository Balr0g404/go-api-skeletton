package email

import "fmt"

// Message is the payload for any email.
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender is the interface implemented by all email providers.
type Sender interface {
	Send(msg Message) error
}

// Config holds provider configuration loaded from env vars.
type Config struct {
	Provider string // "smtp" | "resend" | "noop" (default)
	From     string

	SMTPHost     string
	SMTPPort     int // 587 (STARTTLS) or 465 (TLS)
	SMTPUsername string
	SMTPPassword string

	ResendAPIKey string
}

// New returns a Sender for the given config.
func New(cfg Config) (Sender, error) {
	switch cfg.Provider {
	case "smtp":
		return NewSMTPSender(cfg), nil
	case "resend":
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("email: RESEND_API_KEY is required for resend provider")
		}
		return NewResendSender(cfg.ResendAPIKey, cfg.From), nil
	case "noop", "":
		return &NoopSender{}, nil
	default:
		return nil, fmt.Errorf("email: unknown provider %q (valid: smtp, resend, noop)", cfg.Provider)
	}
}
