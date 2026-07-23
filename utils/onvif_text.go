package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	onvif "github.com/0x524a/onvif-go"
)

const (
	onvifPasswordTextType = "http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText"
)

var (
	reMediaXAddr    = regexp.MustCompile(`(?is)<[^>]*Media[^>]*>.*?<[^>]*XAddr[^>]*>([^<]+)<`)
	reProfileToken  = regexp.MustCompile(`Profiles[^>]*token="([^"]+)"`)
	reSnapshotURI   = regexp.MustCompile(`(?is)<[^>]*Uri[^>]*>([^<]+)<`)
	reManufacturer  = regexp.MustCompile(`(?is)<[^>]*Manufacturer[^>]*>([^<]+)<`)
	reModel         = regexp.MustCompile(`(?is)<[^>]*Model[^>]*>([^<]+)<`)
	reSOAPFaultText = regexp.MustCompile(`(?is)<[^>]*Text[^>]*>([^<]+)<`)
)

func isONVIFNotAuthorized(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "NotAuthorized") || strings.Contains(msg, "not Authorized")
}

func deviceServiceURL(endpoint string, cfg CameraConfig) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint, nil
	}
	host := endpoint
	if cfg.Port > 0 && cfg.Port != 80 {
		if _, _, err := net.SplitHostPort(host); err != nil {
			host = net.JoinHostPort(host, fmt.Sprintf("%d", cfg.Port))
		}
	}
	return "http://" + host + "/onvif/device_service", nil
}

func fixONVIFServiceURL(serviceURL, deviceURL string, cfg CameraConfig) string {
	serviceURL = strings.TrimSpace(serviceURL)
	if serviceURL == "" {
		return mediaServiceURLFromDevice(deviceURL)
	}

	parsed, err := url.Parse(serviceURL)
	if err != nil {
		return serviceURL
	}

	host := parsed.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "0.0.0.0" || host == "::1" {
		device, err := url.Parse(deviceURL)
		if err != nil {
			return serviceURL
		}
		if port := parsed.Port(); port != "" {
			parsed.Host = device.Hostname() + ":" + port
		} else if devicePort := device.Port(); devicePort != "" {
			parsed.Host = device.Hostname() + ":" + devicePort
		} else {
			parsed.Host = device.Hostname()
		}
		return parsed.String()
	}

	if cfg.Host != "" && host != strings.TrimSpace(cfg.Host) {
		cameraHost := strings.TrimSpace(cfg.Host)
		if port := parsed.Port(); port != "" {
			parsed.Host = net.JoinHostPort(cameraHost, port)
		} else {
			parsed.Host = cameraHost
		}
		return parsed.String()
	}

	return serviceURL
}

func mediaServiceURLFromDevice(deviceURL string) string {
	parsed, err := url.Parse(deviceURL)
	if err != nil {
		return deviceURL
	}
	parsed.Path = "/onvif/media_service"
	return parsed.String()
}

func buildONVIFPasswordTextSOAP(user, password, body string) string {
	security := ""
	if user != "" && password != "" {
		security = fmt.Sprintf(
			`<s:Header><wsse:Security s:mustUnderstand="1" xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd"><wsse:UsernameToken><wsse:Username>%s</wsse:Username><wsse:Password Type="%s">%s</wsse:Password></wsse:UsernameToken></wsse:Security></s:Header>`,
			xmlEscape(user),
			onvifPasswordTextType,
			xmlEscape(password),
		)
	}
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope">%s<s:Body>%s</s:Body></s:Envelope>`,
		security,
		body,
	)
}

func xmlEscape(s string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)
	return replacer.Replace(s)
}

func soapPostPasswordText(
	ctx context.Context,
	httpClient *http.Client,
	endpoint, user, password, body string,
) ([]byte, error) {
	payload := buildONVIFPasswordTextSOAP(user, password, body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if fault := parseSOAPFault(respBody); fault != "" {
		return nil, fmt.Errorf("%s", fault)
	}
	return respBody, nil
}

func parseSOAPFault(body []byte) string {
	if !bytes.Contains(body, []byte("Fault")) {
		return ""
	}
	if strings.Contains(string(body), "NotAuthorized") {
		if m := reSOAPFaultText.FindSubmatch(body); len(m) == 2 {
			return strings.TrimSpace(string(m[1]))
		}
		return "Sender not Authorized"
	}
	if m := reSOAPFaultText.FindSubmatch(body); len(m) == 2 {
		text := strings.TrimSpace(string(m[1]))
		if text != "" {
			return text
		}
	}
	return ""
}

func buildONVIFSOAP(body string) string {
	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?><s:Envelope xmlns:s="http://www.w3.org/2003/05/soap-envelope"><s:Body>%s</s:Body></s:Envelope>`,
		body,
	)
}

func soapPost(
	ctx context.Context,
	httpClient *http.Client,
	endpoint, body string,
) ([]byte, error) {
	payload := buildONVIFSOAP(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	if fault := parseSOAPFault(respBody); fault != "" {
		return nil, fmt.Errorf("%s", fault)
	}
	return respBody, nil
}

type onvifSOAPPost func(ctx context.Context, httpClient *http.Client, endpoint, body string) ([]byte, error)

func fetchONVIFSnapshotFallback(
	ctx context.Context,
	cfg CameraConfig,
	deviceEndpoint string,
	priorTiming ONVIFCaptureTiming,
) (ONVIFCaptureResult, error) {
	var lastErr error
	for _, mode := range []struct {
		name string
		post onvifSOAPPost
	}{
		{"digest", soapPost},
		{"text", func(ctx context.Context, httpClient *http.Client, endpoint, body string) ([]byte, error) {
			return soapPostPasswordText(ctx, httpClient, endpoint, cfg.User, cfg.Password, body)
		}},
	} {
		result, err := fetchONVIFSnapshotSOAP(ctx, cfg, deviceEndpoint, priorTiming, mode.post)
		if err == nil {
			return result, nil
		}
		lastErr = fmt.Errorf("onvif %s auth: %w", mode.name, err)
		if !isONVIFNotAuthorized(err) {
			return result, lastErr
		}
	}
	return ONVIFCaptureResult{Timing: priorTiming}, lastErr
}

func fetchONVIFSnapshotSOAP(
	ctx context.Context,
	cfg CameraConfig,
	deviceEndpoint string,
	priorTiming ONVIFCaptureTiming,
	post onvifSOAPPost,
) (ONVIFCaptureResult, error) {
	result := ONVIFCaptureResult{Timing: priorTiming}

	deviceURL, err := deviceServiceURL(deviceEndpoint, cfg)
	if err != nil {
		return result, err
	}

	httpClient := newONVIFHTTPClient(cfg)

	initStart := time.Now()
	capsXML, err := post(ctx, httpClient, deviceURL,
		`<tds:GetCapabilities xmlns:tds="http://www.onvif.org/ver10/device/wsdl"><tds:Category>All</tds:Category></tds:GetCapabilities>`,
	)
	result.Timing.InitMs = time.Since(initStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("initialize: %w", err)
	}

	if infoXML, infoErr := post(ctx, httpClient, deviceURL,
		`<tds:GetDeviceInformation xmlns:tds="http://www.onvif.org/ver10/device/wsdl"/>`,
	); infoErr == nil {
		if m := reManufacturer.FindSubmatch(infoXML); len(m) == 2 {
			result.Manufacturer = strings.TrimSpace(string(m[1]))
		}
		if m := reModel.FindSubmatch(infoXML); len(m) == 2 {
			result.Model = strings.TrimSpace(string(m[1]))
		}
	}

	mediaURL := fixONVIFServiceURL(extractMediaXAddr(capsXML), deviceURL, cfg)

	profilesStart := time.Now()
	profilesXML, err := post(ctx, httpClient, mediaURL,
		`<trt:GetProfiles xmlns:trt="http://www.onvif.org/ver10/media/wsdl"/>`,
	)
	result.Timing.ProfilesMs = time.Since(profilesStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("get profiles: %w", err)
	}

	tokens := reProfileToken.FindAllStringSubmatch(string(profilesXML), -1)
	if len(tokens) == 0 {
		return result, fmt.Errorf("media profile topilmadi")
	}
	if cfg.ProfileIndex >= len(tokens) {
		return result, fmt.Errorf("profile index %d mavjud emas (%d ta)", cfg.ProfileIndex, len(tokens))
	}
	profileToken := strings.TrimSpace(tokens[cfg.ProfileIndex][1])
	if profileToken == "" {
		return result, fmt.Errorf("profile token bo'sh")
	}

	snapshotStart := time.Now()
	snapshotBody := fmt.Sprintf(
		`<trt:GetSnapshotUri xmlns:trt="http://www.onvif.org/ver10/media/wsdl"><trt:ProfileToken>%s</trt:ProfileToken></trt:GetSnapshotUri>`,
		xmlEscape(profileToken),
	)
	snapshotXML, err := post(ctx, httpClient, mediaURL, snapshotBody)
	result.Timing.SnapshotURIMs = time.Since(snapshotStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("get snapshot uri: %w", err)
	}

	snapshotURI := extractSnapshotURI(snapshotXML)
	if snapshotURI == "" {
		return result, fmt.Errorf("snapshot uri bo'sh")
	}

	dlClient, err := onvif.NewClient(
		deviceURL,
		onvif.WithHTTPClient(newONVIFHTTPClient(cfg)),
	)
	if err != nil {
		return result, fmt.Errorf("download client: %w", err)
	}

	fetchStart := time.Now()
	data, err := dlClient.DownloadFile(ctx, snapshotURI)
	result.Timing.FetchImageMs = time.Since(fetchStart).Milliseconds()
	if err != nil {
		return result, fmt.Errorf("download snapshot: %w", err)
	}
	if len(data) < minSnapshotBytes {
		return result, fmt.Errorf("kamera rasmi juda kichik (%d bayt)", len(data))
	}

	result.Data = data
	return result, nil
}

func extractMediaXAddr(capsXML []byte) string {
	if m := reMediaXAddr.FindSubmatch(capsXML); len(m) == 2 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}

func extractSnapshotURI(snapshotXML []byte) string {
	if m := reSnapshotURI.FindSubmatch(snapshotXML); len(m) == 2 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}
