package utils

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCameraTCPReachableOffline(t *testing.T) {
	cfg := CameraConfig{
		Host: "127.0.0.1",
		Port: 1,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := cameraTCPReachable(ctx, cfg); err == nil {
		t.Fatal("expected unreachable camera error")
	}
}

func TestT3CameraUserMessage(t *testing.T) {
	if got := T3CameraUserMessage(fmt.Errorf("kamera javob bermayapti (192.168.1.1:80)")); got == "" {
		t.Fatal("expected message")
	}
}
