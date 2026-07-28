package utils

import (
	"errors"
	"testing"
)

func TestIsDisposableDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "exact disposable domain in email",
			input:    "user@mailinator.com",
			expected: true,
		},
		{
			name:     "subdomain of disposable domain",
			input:    "user@sub.mailinator.com",
			expected: true,
		},
		{
			name:     "nested subdomain of disposable domain",
			input:    "user@a.b.c.mailinator.com",
			expected: true,
		},
		{
			name:     "uppercase disposable domain",
			input:    "user@MAILINATOR.COM",
			expected: true,
		},
		{
			name:     "raw disposable domain string",
			input:    "mailinator.com",
			expected: true,
		},
		{
			name:     "raw disposable subdomain string",
			input:    "sub.mailinator.com",
			expected: true,
		},
		{
			name:     "legitimate email domain",
			input:    "user@gmail.com",
			expected: false,
		},
		{
			name:     "legitimate domain not in list",
			input:    "user@safedomainexample12345.com",
			expected: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDisposableDomain(tt.input)
			if got != tt.expected {
				t.Errorf("IsDisposableDomain(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsValidAndTrustworthyEmail(t *testing.T) {
	t.Run("invalid format", func(t *testing.T) {
		valid, err := IsValidAndTrustworthyEmail("invalid-email-format")
		if valid || err == nil {
			t.Errorf("expected invalid format error, got valid=%v, err=%v", valid, err)
		}
		if !errors.Is(err, ErrInvalidEmailFormat) {
			t.Errorf("expected error wrapping ErrInvalidEmailFormat, got %v", err)
		}
	})

	t.Run("disposable domain", func(t *testing.T) {
		valid, err := IsValidAndTrustworthyEmail("test@mailinator.com")
		if valid || err == nil {
			t.Errorf("expected disposable domain error, got valid=%v, err=%v", valid, err)
		}
		if !errors.Is(err, ErrDisposableDomain) {
			t.Errorf("expected error wrapping ErrDisposableDomain, got %v", err)
		}
	})

	t.Run("valid email with valid MX", func(t *testing.T) {
		// Testing with a known domain with active MX records (gmail.com)
		valid, err := IsValidAndTrustworthyEmail("test@gmail.com")
		if !valid || err != nil {
			t.Errorf("expected valid email, got valid=%v, err=%v", valid, err)
		}
	})

	t.Run("non-existent domain MX lookup failure", func(t *testing.T) {
		valid, err := IsValidAndTrustworthyEmail("test@nonexistent-domain-xyz-1234567890.invalid")
		if valid || err == nil {
			t.Errorf("expected MX lookup error, got valid=%v, err=%v", valid, err)
		}
		if !errors.Is(err, ErrNoMXRecords) {
			t.Errorf("expected error wrapping ErrNoMXRecords, got %v", err)
		}
	})
}
