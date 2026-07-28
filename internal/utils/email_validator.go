package utils

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strings"
)

//go:embed disposable_domains.json
var disposableDomainsJSON []byte

var (
	// ErrInvalidEmailFormat is returned when an email address format is invalid.
	ErrInvalidEmailFormat = errors.New("invalid email format")
	// ErrDisposableDomain is returned when an email domain is determined to be disposable.
	ErrDisposableDomain = errors.New("disposable email domain is not allowed")
	// ErrNoMXRecords is returned when an email domain has no valid MX records.
	ErrNoMXRecords = errors.New("domain has no valid MX records")
)

var disposableDomainsMap map[string]struct{}

func init() {
	var domains []string
	if err := json.Unmarshal(disposableDomainsJSON, &domains); err != nil {
		panic(fmt.Sprintf("failed to parse disposable_domains.json: %v", err))
	}

	disposableDomainsMap = make(map[string]struct{}, len(domains))
	for _, d := range domains {
		domain := strings.ToLower(strings.TrimSpace(d))
		if domain != "" {
			disposableDomainsMap[domain] = struct{}{}
		}
	}
}

func IsDisposableDomain(email string) bool {
	email = strings.TrimSpace(email)
	if email == "" {
		return false
	}

	if addr, err := mail.ParseAddress(email); err == nil {
		email = addr.Address
	}

	domain := email
	if atIdx := strings.LastIndex(email, "@"); atIdx != -1 {
		domain = email[atIdx+1:]
	}

	domain = strings.ToLower(strings.Trim(domain, " ."))
	if domain == "" {
		return false
	}

	curr := domain
	for curr != "" {
		if _, ok := disposableDomainsMap[curr]; ok {
			return true
		}
		dotIdx := strings.IndexByte(curr, '.')
		if dotIdx == -1 {
			break
		}
		curr = curr[dotIdx+1:]
	}

	return false
}

func IsValidAndTrustworthyEmail(email string) (bool, error) {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidEmailFormat, err)
	}

	atIdx := strings.LastIndex(addr.Address, "@")
	if atIdx <= 0 || atIdx == len(addr.Address)-1 {
		return false, ErrInvalidEmailFormat
	}

	domain := strings.ToLower(strings.Trim(addr.Address[atIdx+1:], " ."))

	if IsDisposableDomain(domain) {
		return false, ErrDisposableDomain
	}

	mxs, err := net.LookupMX(domain)
	if err != nil || len(mxs) == 0 {
		if err != nil {
			return false, fmt.Errorf("%w: %v", ErrNoMXRecords, err)
		}
		return false, ErrNoMXRecords
	}

	return true, nil
}
