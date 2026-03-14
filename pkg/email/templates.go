package email

import "fmt"

// Welcome returns a welcome message sent after successful registration.
func Welcome(firstName, toEmail string) Message {
	return Message{
		To:      toEmail,
		Subject: "Welcome!",
		HTML: fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2>Welcome, %s!</h2>
  <p>Your account has been created successfully.</p>
  <p>You can now log in and start using the API.</p>
</body>
</html>`, firstName),
		Text: fmt.Sprintf("Welcome, %s! Your account has been created successfully.", firstName),
	}
}

// PasswordReset returns the password-reset email containing the reset link.
func PasswordReset(firstName, resetURL string) Message {
	return Message{
		To:      "",
		Subject: "Reset your password",
		HTML: fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:600px;margin:0 auto;padding:24px">
  <h2>Password reset request</h2>
  <p>Hi %s,</p>
  <p>Click the link below to reset your password. The link expires in 1 hour.</p>
  <p><a href="%s" style="background:#2563eb;color:#fff;padding:12px 24px;border-radius:6px;text-decoration:none">Reset password</a></p>
  <p>If you did not request a password reset, you can ignore this email.</p>
</body>
</html>`, firstName, resetURL),
		Text: fmt.Sprintf("Hi %s,\n\nReset your password: %s\n\nThis link expires in 1 hour.", firstName, resetURL),
	}
}
