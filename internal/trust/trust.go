// Package trust decides which SSH logins are trusted: those from addresses
// or networks on the whitelist, and those by whitelisted users such as a CI
// deploy account that connects from ever-changing addresses.
//
// Trusted logins are still reported, only silently, so a stolen Telegram
// account cannot hide a login by whitelisting it.
package trust

import (
	"errors"
	"net"
	"regexp"
	"strings"
)

// A Linux user name, as accepted by useradd.
var userRe = regexp.MustCompile(`^[a-z_][a-z0-9_.-]{0,31}$`)

// Normalize validates a whitelist entry and returns it in canonical form:
// an IP address, a CIDR network, or a user name.
func Normalize(entry string) (string, error) {
	entry = strings.TrimSpace(entry)
	if strings.Contains(entry, "/") {
		_, n, err := net.ParseCIDR(entry)
		if err != nil {
			return "", errors.New("invalid network")
		}
		return n.String(), nil
	}
	if ip := net.ParseIP(entry); ip != nil {
		return ip.String(), nil
	}
	if userRe.MatchString(entry) {
		return entry, nil
	}
	return "", errors.New("not an IP address, network or user name")
}

// IsUser reports whether a normalized entry is a user name.
func IsUser(entry string) bool {
	return !strings.Contains(entry, "/") && net.ParseIP(entry) == nil
}

// Match reports whether a login by user from host matches an entry.
func Match(entries []string, user, host string) bool {
	ip := net.ParseIP(host)
	for _, e := range entries {
		if IsUser(e) {
			if e == user {
				return true
			}
			continue
		}
		if ip == nil {
			continue
		}
		if strings.Contains(e, "/") {
			if _, n, err := net.ParseCIDR(e); err == nil && n.Contains(ip) {
				return true
			}
		} else if net.ParseIP(e).Equal(ip) {
			return true
		}
	}
	return false
}
