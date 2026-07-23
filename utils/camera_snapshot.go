package utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	onvif "github.com/0x524a/onvif-go"
)

const minSnapshotBytes = 1024

type CameraConfig struct {
	Enabled       bool
	Host          string
	Port          int
	DeviceService string
	User          string
	Password      string
	ProfileIndex  int
	Timeout       time.Duration
}

type ONVIFCaptureTiming struct {
	InitMs         int64
	ProfilesMs     int64
	SnapshotURIMs  int64
	FetchImageMs   int64
}

type ONVIFCaptureResult struct {
	Data         []byte
	Manufacturer string
	Model        string
	Timing       ONVIFCaptureTiming
}

type SaveT3ScanPhotoResult struct {
	FilePath   string
	FullPath   string
	CapturedAt time.Time
}

func LoadCameraConfigFromEnv() CameraConfig {
	cfg := CameraConfig{
		Enabled:      strings.EqualFold(strings.TrimSpace(os.Getenv("T3_CAMERA_ENABLED")), "true"),
		Host:         strings.TrimSpace(os.Getenv("T3_CAMERA_HOST")),
		Port:         80,
		User:         strings.TrimSpace(os.Getenv("T3_CAMERA_USER")),
		Password:     os.Getenv("T3_CAMERA_PASSWORD"),
		ProfileIndex: 0,
		Timeout:      15 * time.Second,
	}

	if port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("T3_CAMERA_PORT"))); err == nil && port > 0 {
		cfg.Port = port
	}
	if idx, err := strconv.Atoi(strings.TrimSpace(os.Getenv("T3_CAMERA_PROFILE_INDEX"))); err == nil && idx >= 0 {
		cfg.ProfileIndex = idx
	}
	if ms, err := strconv.Atoi(strings.TrimSpace(os.Getenv("T3_CAMERA_TIMEOUT_MS"))); err == nil && ms > 0 {
		cfg.Timeout = time.Duration(ms) * time.Millisecond
	}
	cfg.DeviceService = strings.TrimSpace(os.Getenv("T3_CAMERA_DEVICE_SERVICE"))

	return cfg
}

func cameraPingTimeout() time.Duration {
	if ms, err := strconv.Atoi(strings.TrimSpace(os.Getenv("T3_CAMERA_PING_MS"))); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return 2 * time.Second
}

func cameraTCPReachable(ctx context.Context, cfg CameraConfig) error {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return fmt.Errorf("kamera host sozlanmagan")
	}
	port := cfg.Port
	if port <= 0 {
		port = 80
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		host = h
		if portNum, err := strconv.Atoi(p); err == nil && portNum > 0 {
			port = portNum
		}
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := net.Dialer{Timeout: cameraPingTimeout()}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("kamera javob bermayapti (%s)", addr)
	}
	_ = conn.Close()
	return nil
}

// T3CameraUserMessage returns a short message for UI when capture fails.
func T3CameraUserMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "javob bermayapti"):
		return "Kamera ulanmagan — surat olinmadi"
	case strings.Contains(msg, "o'chirilgan"):
		return "Kamera o'chirilgan — surat olinmadi"
	case strings.Contains(msg, "NotAuthorized"):
		return "Kamera login xato — surat olinmadi"
	default:
		return "Kamera surati olinmadi"
	}
}

func cameraONVIPEndpoint(cfg CameraConfig) (string, error) {
	if cfg.DeviceService != "" {
		return cfg.DeviceService, nil
	}
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return "", fmt.Errorf("kamera host sozlanmagan")
	}
	if cfg.Port > 0 && cfg.Port != 80 {
		if _, _, err := net.SplitHostPort(host); err != nil {
			host = net.JoinHostPort(host, strconv.Itoa(cfg.Port))
		}
	}
	return host, nil
}

func FetchONVIFSnapshot(ctx context.Context, cfg CameraConfig) (ONVIFCaptureResult, error) {
	result := ONVIFCaptureResult{}
	if !cfg.Enabled {
		return result, fmt.Errorf("kamera o'chirilgan (T3_CAMERA_ENABLED=false)")
	}

	endpoint, err := cameraONVIPEndpoint(cfg)
	if err != nil {
		return result, err
	}

	if err := cameraTCPReachable(ctx, cfg); err != nil {
		return result, err
	}

	httpClient := newONVIFHTTPClient(cfg)

	client, err := onvif.NewClient(
		endpoint,
		onvif.WithHTTPClient(httpClient),
	)
	if err != nil {
		return result, fmt.Errorf("onvif client: %w", err)
	}

	initStart := time.Now()
	if err := client.Initialize(ctx); err != nil {
		// Dahua often needs WS-Security in addition to HTTP Digest.
		client, err = onvif.NewClient(
			endpoint,
			onvif.WithCredentials(cfg.User, cfg.Password),
			onvif.WithHTTPClient(httpClient),
		)
		if err != nil {
			return result, fmt.Errorf("onvif client: %w", err)
		}
		if err := client.Initialize(ctx); err != nil {
			return result, fmt.Errorf("onvif initialize: %w", err)
		}
	}
	result.Timing.InitMs = time.Since(initStart).Milliseconds()

	if info, err := client.GetDeviceInformation(ctx); err == nil && info != nil {
		result.Manufacturer = info.Manufacturer
		result.Model = info.Model
	}

	profilesStart := time.Now()
	profiles, err := client.GetProfiles(ctx)
	result.Timing.ProfilesMs = time.Since(profilesStart).Milliseconds()
	if err != nil {
		if isONVIFNotAuthorized(err) {
			textResult, textErr := fetchONVIFSnapshotFallback(ctx, cfg, endpoint, result.Timing)
			if textErr == nil {
				if result.Manufacturer != "" && textResult.Manufacturer == "" {
					textResult.Manufacturer = result.Manufacturer
				}
				if result.Model != "" && textResult.Model == "" {
					textResult.Model = result.Model
				}
				return textResult, nil
			}
		}
		return result, fmt.Errorf("onvif get profiles: %w", err)
	}
	if len(profiles) == 0 {
		return result, fmt.Errorf("onvif: media profile topilmadi")
	}
	if cfg.ProfileIndex >= len(profiles) {
		return result, fmt.Errorf("onvif: profile index %d mavjud emas (%d ta)", cfg.ProfileIndex, len(profiles))
	}

	profileToken := strings.TrimSpace(profiles[cfg.ProfileIndex].Token)
	if profileToken == "" {
		return result, fmt.Errorf("onvif: profile token bo'sh")
	}

	snapshotStart := time.Now()
	snapshotURI, err := client.GetSnapshotURI(ctx, profileToken)
	result.Timing.SnapshotURIMs = time.Since(snapshotStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("onvif get snapshot uri: %w", err)
	}
	if snapshotURI == nil || strings.TrimSpace(snapshotURI.URI) == "" {
		return result, fmt.Errorf("onvif: snapshot uri bo'sh")
	}

	fetchStart := time.Now()
	data, err := client.DownloadFile(ctx, snapshotURI.URI)
	result.Timing.FetchImageMs = time.Since(fetchStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("onvif download snapshot: %w", err)
	}
	if len(data) < minSnapshotBytes {
		return result, fmt.Errorf("kamera rasmi juda kichik (%d bayt)", len(data))
	}

	result.Data = data
	return result, nil
}

func SanitizeT3ScanLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "test"
	}
	var b strings.Builder
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "test"
	}
	return b.String()
}

func SaveT3ScanPhoto(label string, data []byte, capturedAt time.Time) (SaveT3ScanPhotoResult, error) {
	result := SaveT3ScanPhotoResult{CapturedAt: capturedAt}
	if capturedAt.IsZero() {
		result.CapturedAt = time.Now()
		capturedAt = result.CapturedAt
	}

	safeLabel := SanitizeT3ScanLabel(label)
	dir := filepath.Join(
		"uploads", "t3-scans",
		capturedAt.Format("2006"),
		capturedAt.Format("01"),
		capturedAt.Format("02"),
	)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return result, err
	}

	photoData := data
	if annotated, err := annotateT3ScanPhotoJPEG(data, safeLabel); err == nil {
		photoData = annotated
	}

	filename := fmt.Sprintf("%s.jpg", safeLabel)
	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, photoData, 0644); err != nil {
		return result, err
	}

	result.FullPath = fullPath
	result.FilePath = "/" + filepath.ToSlash(fullPath)
	return result, nil
}
