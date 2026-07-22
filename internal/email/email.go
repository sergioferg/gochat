package email

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wneessen/go-mail"
)

func SendEmail(to, subject, body string) error {
	m := mail.NewMsg()
	if err := m.From("test@gochat.com"); err != nil {
		logrus.Error(fmt.Sprintf("failed to set From address: %s", err))
		return err
	}
	if err := m.To(to); err != nil {
		logrus.Error(fmt.Sprintf("failed to set To address: %s", err))
		return err
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextPlain, body)

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
