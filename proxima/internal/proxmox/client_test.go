package proxmox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func TestPasswordAuthTicket(t *testing.T) {
	var gotUser, gotPassword string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api2/json/access/ticket" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotUser = r.Form.Get("username")
		gotPassword = r.Form.Get("password")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{
				"ticket":              "PVE:ticket-value",
				"CSRFPreventionToken": "csrf-token-value",
			},
		})
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	cfg := config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth: config.Auth{
			User:     "root@pam",
			Password: "secret",
		},
		InsecureTLS: true,
	}

	client, err := proxmox.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if gotUser != "root@pam" || gotPassword != "secret" {
		t.Fatalf("form user=%q password=%q", gotUser, gotPassword)
	}
	if client.Ticket() != "PVE:ticket-value" {
		t.Fatalf("ticket = %q", client.Ticket())
	}
	if client.CSRFToken() != "csrf-token-value" {
		t.Fatalf("csrf = %q", client.CSRFToken())
	}
}

func TestTokenAuthHeader(t *testing.T) {
	ticketCalled := false
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api2/json/access/ticket" {
			ticketCalled = true
			http.Error(w, "should not ticket", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth: config.Auth{
			TokenID:     "root@pam!proxima",
			TokenSecret: "uuid-secret",
		},
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if ticketCalled {
		t.Fatal("token auth must not call /access/ticket")
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api2/json/version", nil)
	if err != nil {
		t.Fatal(err)
	}
	client.ApplyAuth(req)
	want := "PVEAPIToken=root@pam!proxima=uuid-secret"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
	if client.Ticket() != "" {
		t.Fatalf("ticket should be empty for token auth, got %q", client.Ticket())
	}
}

func TestInsecureTLSFlagPropagates(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]string{
				"ticket":              "PVE:t",
				"CSRFPreventionToken": "csrf",
			},
		})
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	secure, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth:        config.Auth{User: "root@pam", Password: "secret"},
		InsecureTLS: false,
	})
	// TLS server uses unknown CA; without InsecureTLS NewClient auth should fail to dial.
	if err == nil {
		t.Fatal("expected TLS failure when InsecureTLS is false")
	}
	_ = secure

	insecure, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth:        config.Auth{User: "root@pam", Password: "secret"},
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewClient insecure: %v", err)
	}
	if !insecure.TLSInsecureSkipVerify() {
		t.Fatal("InsecureSkipVerify should be true when InsecureTLS is set")
	}
}

func TestAuthFailure(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api2/json/access/ticket" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":"auth"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	_, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth:        config.Auth{User: "root@pam", Password: "wrong"},
		InsecureTLS: true,
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if err.Error() != "proxmox authentication failed" {
		t.Fatalf("error = %q, want %q", err.Error(), "proxmox authentication failed")
	}
}

func TestNodeStatus(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"ticket":              "PVE:t",
					"CSRFPreventionToken": "csrf",
				},
			})
		case "/api2/json/nodes/pve/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"uptime": 123,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth:        config.Auth{User: "root@pam", Password: "secret"},
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	name, err := client.NodeStatus(context.Background())
	if err != nil {
		t.Fatalf("NodeStatus: %v", err)
	}
	if name != "pve" {
		t.Fatalf("node = %q, want pve", name)
	}
}

func TestVersionParse(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"ticket":              "PVE:t",
					"CSRFPreventionToken": "csrf",
				},
			})
		case "/api2/json/version":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"version": "8.2.2",
					"release": "8.2",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	host, port := hostPort(t, srv.URL)
	client, err := proxmox.NewClient(config.Config{
		ProxmoxHost: host,
		ProxmoxPort: port,
		Node:        "pve",
		Auth:        config.Auth{User: "root@pam", Password: "secret"},
		InsecureTLS: true,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ver, err := client.Version(context.Background())
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if ver != "8.2.2" {
		t.Fatalf("version = %q, want 8.2.2", ver)
	}
}

func hostPort(t *testing.T, raw string) (string, int) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return u.Hostname(), port
}
