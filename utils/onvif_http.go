package utils

import (
	"net/http"

	"github.com/icholy/digest"
)

func newONVIFHTTPClient(cfg CameraConfig) *http.Client {
	if cfg.User == "" || cfg.Password == "" {
		return &http.Client{Timeout: cfg.Timeout}
	}
	return &http.Client{
		Timeout: cfg.Timeout,
		Transport: &digest.Transport{
			Username: cfg.User,
			Password: cfg.Password,
		},
	}
}
