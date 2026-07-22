package mailer

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

func SendEmail(to, username, actionURL string) error {
	m := mail.NewMsg()
	if err := m.From("test@gochat.com"); err != nil {
		logrus.Error(fmt.Sprintf("failed to set From address: %s", err))
		return err
	}
	if err := m.To(to); err != nil {
		logrus.Error(fmt.Sprintf("failed to set To address: %s", err))
		return err
	}
	m.Subject("[Gochat] Verify your account")

	plainText := fmt.Sprintf("Welcome %s! Click here to confirm your account: %s", username, actionURL)
	m.SetBodyString(mail.TypeTextPlain, plainText)

	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; background-color: #f4f4f5; padding: 20px;">
			<div style="max-width: 600px; margin: 0 auto; background: #ffffff; padding: 20px; border-radius: 8px;">
				Hello %s!
				<br><br>
				We received a request to verify an account with gochat ID: <strong>%s</strong><br>
				To verify your account, click the link below:
				<br><br>
				<a href="%s" style="background:#00acd7;padding:10px 20px;color:#fff;font-size:.85rem;text-decoration:none;display:inline-block;border-radius:4px;margin:0 auto" rel="noopener noreferrer" target="_blank">
					Verify Account
				</a>
				<br><br>
				If you did not make this request, your email address may have been
				entered by mistake and you can safely disregard this email.
				<br><br>
				If you have any questions or concerns, please contact us at <a href="mailto:support@gochat.com" target="_blank">support@gochat.com</a>.
				<br><br>
				Thank you,<br>
				The Gochat Team
			</div>
		</body>
		</html>
	`, username, username, actionURL)

	m.AddAlternativeString(mail.TypeTextHTML, htmlContent)

	client, err := mail.NewClient(
		"localhost",
		mail.WithPort(1025),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		logrus.Warn("failed to create SMTP client:", err)
		return err // FIX: Added return to prevent nil-pointer panic below!
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		logrus.Warn("failed to send mail:", err)
		return err
	}

	logrus.Info("Email delivered to SMTP broker successfully!")
	return nil
}
