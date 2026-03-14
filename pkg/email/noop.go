package email

// NoopSender discards all emails. Used in development and tests.
type NoopSender struct{}

func (n *NoopSender) Send(_ Message) error { return nil }
