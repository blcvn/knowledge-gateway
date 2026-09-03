package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8000"
	defaultTimeout = 20 * time.Second
)

type config struct {
	BaseURL       string
	AppName       string
	Description   string
	Owner         string
	Timeout       time.Duration
	ReuseExisting bool
	Format        string
}

type registerResult struct {
	AppID  string `json:"app_id"`
	Status string `json:"status,omitempty"`
	Reused bool   `json:"reused_existing"`
}

type listAppsResponse struct {
	Apps []appInfo `json:"apps"`
}

type appInfo struct {
	AppIDSnake string `json:"app_id"`
	AppIDCamel string `json:"appId"`
	AppName    string `json:"app_name"`
	AppNameAlt string `json:"appName"`
	Owner      string `json:"owner"`
	Status     string `json:"status"`
}

type createAppRequest struct {
	AppName     string `json:"app_name"`
	Description string `json:"description,omitempty"`
	Owner       string `json:"owner"`
}

type createAppResponse struct {
	AppIDSnake string `json:"app_id"`
	AppIDCamel string `json:"appId"`
	Status     string `json:"status"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "kg-app-register failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := parseFlags()
	if err := validateConfig(&cfg); err != nil {
		return err
	}

	client := &http.Client{Timeout: cfg.Timeout}

	if cfg.ReuseExisting {
		app, found, err := findExistingApp(client, cfg)
		if err != nil {
			return err
		}
		if found {
			return printResult(cfg.Format, registerResult{
				AppID:  app.AppID(),
				Status: strings.TrimSpace(app.Status),
				Reused: true,
			})
		}
	}

	created, err := createApp(client, cfg)
	if err != nil {
		return err
	}

	return printResult(cfg.Format, registerResult{
		AppID:  created.AppID(),
		Status: strings.TrimSpace(created.Status),
		Reused: false,
	})
}

func parseFlags() config {
	var cfg config
	flag.StringVar(&cfg.BaseURL, "base-url", envOr("KGS_BASE_URL", defaultBaseURL), "KGS HTTP base URL")
	flag.StringVar(&cfg.AppName, "app-name", "", "Application name to register (required)")
	flag.StringVar(&cfg.Description, "description", "", "Application description")
	flag.StringVar(&cfg.Owner, "owner", "", "Application owner, e.g. system@internal (required)")
	flag.DurationVar(&cfg.Timeout, "timeout", defaultTimeout, "HTTP timeout per request")
	flag.BoolVar(&cfg.ReuseExisting, "reuse-existing", true, "Return existing app_id if same app_name + owner already exists")
	flag.StringVar(&cfg.Format, "format", "app_id", "Output format: app_id | env | json")
	flag.Parse()
	return cfg
}

func validateConfig(cfg *config) error {
	cfg.AppName = strings.TrimSpace(cfg.AppName)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.Owner = strings.TrimSpace(cfg.Owner)
	cfg.Format = strings.ToLower(strings.TrimSpace(cfg.Format))

	if cfg.AppName == "" {
		return errors.New("missing required -app-name")
	}
	if cfg.Owner == "" {
		return errors.New("missing required -owner")
	}
	if cfg.Timeout <= 0 {
		return errors.New("timeout must be > 0")
	}
	switch cfg.Format {
	case "app_id", "env", "json":
	default:
		return fmt.Errorf("invalid -format %q, expected app_id|env|json", cfg.Format)
	}

	baseURL, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return err
	}
	cfg.BaseURL = baseURL
	return nil
}

func normalizeBaseURL(raw string) (string, error) {
	base := strings.TrimSpace(raw)
	if base == "" {
		return "", errors.New("base-url must not be empty")
	}
	if !strings.Contains(base, "://") {
		base = "http://" + base
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base-url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("invalid base-url %q", raw)
	}
	return strings.TrimRight(base, "/"), nil
}

func findExistingApp(client *http.Client, cfg config) (appInfo, bool, error) {
	endpoint := cfg.BaseURL + "/v1/apps"
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return appInfo{}, false, fmt.Errorf("build list apps request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return appInfo{}, false, fmt.Errorf("list apps request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return appInfo{}, false, fmt.Errorf("list apps failed (status=%d): %s", resp.StatusCode, compactBody(body))
	}

	var payload listAppsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return appInfo{}, false, fmt.Errorf("decode list apps response: %w", err)
	}

	appName := normalizeMatch(cfg.AppName)
	owner := normalizeMatch(cfg.Owner)
	for _, app := range payload.Apps {
		if app.AppID() == "" {
			continue
		}
		if normalizeMatch(app.Name()) == appName && normalizeMatch(app.Owner) == owner {
			return app, true, nil
		}
	}
	return appInfo{}, false, nil
}

func createApp(client *http.Client, cfg config) (createAppResponse, error) {
	reqBody := createAppRequest{
		AppName:     cfg.AppName,
		Description: cfg.Description,
		Owner:       cfg.Owner,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return createAppResponse{}, fmt.Errorf("marshal create app payload: %w", err)
	}

	endpoint := cfg.BaseURL + "/v1/apps"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return createAppResponse{}, fmt.Errorf("build create app request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return createAppResponse{}, fmt.Errorf("create app request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return createAppResponse{}, fmt.Errorf("create app failed (status=%d): %s", resp.StatusCode, compactBody(body))
	}

	var payload createAppResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return createAppResponse{}, fmt.Errorf("decode create app response: %w", err)
	}
	if payload.AppID() == "" {
		return createAppResponse{}, fmt.Errorf("create app succeeded but response missing app_id: %s", compactBody(body))
	}
	return payload, nil
}

func printResult(format string, result registerResult) error {
	switch format {
	case "app_id":
		fmt.Println(result.AppID)
		return nil
	case "env":
		fmt.Printf("KG_APP_ID=%s\n", result.AppID)
		return nil
	case "json":
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("encode json result: %w", err)
		}
		fmt.Println(string(data))
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func (a appInfo) AppID() string {
	if strings.TrimSpace(a.AppIDSnake) != "" {
		return strings.TrimSpace(a.AppIDSnake)
	}
	return strings.TrimSpace(a.AppIDCamel)
}

func (a appInfo) Name() string {
	if strings.TrimSpace(a.AppName) != "" {
		return strings.TrimSpace(a.AppName)
	}
	return strings.TrimSpace(a.AppNameAlt)
}

func (r createAppResponse) AppID() string {
	if strings.TrimSpace(r.AppIDSnake) != "" {
		return strings.TrimSpace(r.AppIDSnake)
	}
	return strings.TrimSpace(r.AppIDCamel)
}

func normalizeMatch(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func compactBody(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return "empty response"
	}
	return text
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
