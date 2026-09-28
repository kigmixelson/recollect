package query

import (
	"fmt"
	"net/url"
	"strings"
)

func NormalizeHost(host, scheme string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is empty")
	}
	scheme = strings.ToLower(strings.TrimSpace(scheme))
	switch scheme {
	case "", "http", "https":
	default:
		return "", fmt.Errorf("unsupported scheme %q (http or https)", scheme)
	}

	if !strings.Contains(host, "://") {
		if scheme == "" {
			scheme = "https"
		}
		host = scheme + "://" + host
		scheme = ""
	}

	u, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("invalid host %q: %w", host, err)
	}
	if scheme != "" {
		u.Scheme = scheme
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported host scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid host %q", host)
	}
	return u.Scheme + "://" + u.Host, nil
}
