package email

import (
	"context"
	"fmt"
	"log"
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
	plainText := "Welcome " + username + "! Click here to confirm your account:" + actionURL
	m.SetBodyString(mail.TypeTextPlain, plainText)

	htmlContent := `
		<!DOCTYPE html>
		<html>
		<body style="font-family: Arial, sans-serif; background-color: #f4f4f5; padding: 20px;">
                Hello ` + username + `!
                <br>
                <br>
                We received a request to verify an account with gochat ID: <strong>` + username + `</strong><br>
                To verify your account, click the link below:
                <br>
                <br>
                <a href="` + actionURL + `" style="background:#00acd7;padding:10px 20px;color:#fff;font-size:.85rem;text-decoration:none;display:inline-block;border-radius:4px;margin:0 auto" rel="noopener noreferrer" target="_blank" data-saferedirecturl="https://www.google.com/url?q=` + actionURL + `">
                    Reset Password
                </a>
                <br>
                <br>
                If you did not make this request, your email address may have been
                entered by mistake and you can safely disregard this email.
                <br>
                <br>
                If you have any questions or concerns, please contact us at <a href="mailto:support@gochat.com" target="_blank">support@gochat.com</a>.
                <br>
                <br>
                Thank you,
                <br>
                The Gochat Team

			</div>
		</body>
		</html>
		`

	m.AddAlternativeString(mail.TypeTextHTML, htmlContent)

	client, err := mail.NewClient(
		"localhost",
		mail.WithPort(1025),
		mail.WithTLSPolicy(mail.NoTLS),
	)
	if err != nil {
		log.Fatalf("failed to create SMTP client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := client.DialAndSendWithContext(ctx, m); err != nil {
		log.Fatalf("failed to send mail: %v", err)
		return err
	}

	logrus.Info("Email delivered to SMTP broker successfully!")
	return nil
}
