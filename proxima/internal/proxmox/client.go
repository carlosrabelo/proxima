package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"proxima/proxima/internal/config"
)

// Client talks to the Proxmox VE HTTP API.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	node        string
	auth        config.Auth
	ticket      string
	csrfToken   string
	insecureTLS bool
}

type ticketResponse struct {
	Data struct {
		Ticket              string `json:"ticket"`
		CSRFPreventionToken string `json:"CSRFPreventionToken"`
	} `json:"data"`
}

// NewClient builds an authenticated Proxmox API client from cfg.
func NewClient(cfg config.Config) (*Client, error) {
	port := cfg.ProxmoxPort
	if port == 0 {
		port = 8006
	}
	baseURL := fmt.Sprintf("https://%s:%d/api2/json", cfg.ProxmoxHost, port)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureTLS}, //nolint:gosec // intentional when InsecureTLS
	}
	c := &Client{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		baseURL:     baseURL,
		node:        cfg.Node,
		auth:        cfg.Auth,
		insecureTLS: cfg.InsecureTLS,
	}

	hasPassword := cfg.Auth.User != "" || cfg.Auth.Password != ""
	hasToken := cfg.Auth.TokenID != "" || cfg.Auth.TokenSecret != ""
	switch {
	case hasPassword && !hasToken:
		if err := c.authenticatePassword(context.Background()); err != nil {
			return nil, err
		}
	case hasToken && !hasPassword:
		// Token auth attaches Authorization on each request; no ticket call.
	}

	return c, nil
}

// ApplyAuth sets Proxmox authentication on req (cookie/CSRF or API token header).
func (c *Client) ApplyAuth(req *http.Request) {
	if c.auth.TokenID != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s=%s", c.auth.TokenID, c.auth.TokenSecret))
		return
	}
	if c.ticket != "" {
		req.AddCookie(&http.Cookie{Name: "PVEAuthCookie", Value: c.ticket})
		if req.Method != http.MethodGet && req.Method != http.MethodHead && c.csrfToken != "" {
			req.Header.Set("CSRFPreventionToken", c.csrfToken)
		}
	}
}

func (c *Client) authenticatePassword(ctx context.Context) error {
	form := url.Values{}
	form.Set("username", c.auth.User)
	form.Set("password", c.auth.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/access/ticket", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("proxmox api unreachable: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("proxmox api unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("proxmox api: invalid response")
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("proxmox authentication failed")
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return fmt.Errorf("proxmox api error: status %d", resp.StatusCode)
	}

	var parsed ticketResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return fmt.Errorf("proxmox api: invalid response")
	}
	if parsed.Data.Ticket == "" {
		return fmt.Errorf("proxmox api: invalid response")
	}

	c.ticket = parsed.Data.Ticket
	c.csrfToken = parsed.Data.CSRFPreventionToken
	return nil
}

// NodeStatus checks that the configured node is reachable and returns its name.
func (c *Client) NodeStatus(ctx context.Context) (string, error) {
	_, err := c.get(ctx, "/nodes/"+url.PathEscape(c.node)+"/status")
	if err != nil {
		return "", err
	}
	return c.node, nil
}

// Version returns the Proxmox API version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	body, err := c.get(ctx, "/version")
	if err != nil {
		return "", err
	}
	var parsed struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.Data.Version == "" {
		return "", fmt.Errorf("proxmox api: invalid response")
	}
	return parsed.Data.Version, nil
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("proxmox api unreachable: %w", err)
	}
	c.ApplyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("proxmox api unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("proxmox api: invalid response")
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("proxmox authentication failed")
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("proxmox api error: status %d", resp.StatusCode)
	}
	return body, nil
}

// Ticket returns the PVE auth ticket (tests / later requests).
func (c *Client) Ticket() string { return c.ticket }

// CSRFToken returns the CSRF prevention token.
func (c *Client) CSRFToken() string { return c.csrfToken }

// TLSInsecureSkipVerify reports whether TLS verification is disabled.
func (c *Client) TLSInsecureSkipVerify() bool {
	tr, ok := c.httpClient.Transport.(*http.Transport)
	if !ok || tr.TLSClientConfig == nil {
		return c.insecureTLS
	}
	return tr.TLSClientConfig.InsecureSkipVerify
}
