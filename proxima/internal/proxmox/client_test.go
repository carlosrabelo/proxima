package proxmox_test

import (
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
