package utils

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var connPortRE = regexp.MustCompile(`(?i)\bport\s*=\s*['"]?\d+['"]?`)

func DBConnString() (string, error) {
	connString := strings.TrimSpace(os.Getenv("CONN_STRING"))
	if connString == "" {
		return "", errors.New("CONN_STRING is empty. Set it in .env or environment variables")
	}

	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	if port == "" {
		port = "5432"
	}
	if _, err := net.LookupPort("tcp", port); err != nil {
		return "", fmt.Errorf("invalid DB_PORT %q: %w", port, err)
	}

	return applyDBPort(connString, port), nil
}

func applyDBPort(connString, port string) string {
	lower := strings.ToLower(connString)
	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		u, err := url.Parse(connString)
		if err != nil {
			return connString
		}
		u.Host = net.JoinHostPort(u.Hostname(), port)
		return u.String()
	}

	if connPortRE.MatchString(connString) {
		return connPortRE.ReplaceAllString(connString, "port="+port)
	}

	return strings.TrimSpace(connString) + " port=" + port
}
