package main

import (
	"time"

	"github.com/sergioferg/gochat/internal/email"
)

func main() {
	go func() {
		email.SendEmail("anyone@example.com", "cool message", "This is a coooool message!")
	}()
	time.Sleep(30 * time.Second)
}
