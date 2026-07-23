package utils

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off", "disable", "disabled":
		return false
	default:
		return fallback
	}
}

func envIntOrDefault(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// VtmMSSQLConnString builds a sqlserver:// connection URL for the lab VTM database.
// Values are read from .env (VTM_MSSQL_*).
func VtmMSSQLConnString() string {
	host := envOrDefault("VTM_MSSQL_HOST", "192.168.5.95")
	dbName := envOrDefault("VTM_MSSQL_DB", "Vtm")
	user := envOrDefault("VTM_MSSQL_USER", "sa")
	password := os.Getenv("VTM_MSSQL_PASSWORD")
	encrypt := "disable"
	if envBoolOrDefault("VTM_MSSQL_ENCRYPT", false) {
		encrypt = "true"
	}
	trustCert := "true"
	if !envBoolOrDefault("VTM_MSSQL_TRUST_SERVER_CERTIFICATE", true) {
		trustCert = "false"
	}
	timeoutSec := envIntOrDefault("VTM_MSSQL_TIMEOUT_SEC", 10)
	timeout := strconv.Itoa(timeoutSec)

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(user, password),
		Host:   host,
	}
	q := url.Values{}
	q.Set("database", dbName)
	q.Set("encrypt", encrypt)
	q.Set("TrustServerCertificate", trustCert)
	q.Set("dial timeout", timeout)
	q.Set("connection timeout", timeout)
	u.RawQuery = q.Encode()
	return u.String()
}

// VtmMSSQLHostForLog returns host/db for logging without credentials.
func VtmMSSQLHostForLog() string {
	host := envOrDefault("VTM_MSSQL_HOST", "192.168.5.95")
	dbName := envOrDefault("VTM_MSSQL_DB", "Vtm")
	return fmt.Sprintf("%s/%s", host, dbName)
}
