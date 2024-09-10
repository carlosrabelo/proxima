package proxmox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func TestPickFirstNonLoopbackIPv4(t *testing.T) {
	raw := []byte(`{
  "result": [
    {
      "name": "lo",
      "ip-addresses": [
        {"ip-address": "127.0.0.1", "ip-address-type": "ipv4", "prefix": 8}
      ]
    },
    {
      "name": "eth0",
      "ip-addresses": [
        {"ip-address": "10.0.0.42", "ip-address-type": "ipv4", "prefix": 24},
        {"ip-address": "fe80::1", "ip-address-type": "ipv6", "prefix": 64}
      ]
    }
  ]
}`)
	got, ok := proxmox.PickFirstNonLoopbackIPv4(raw)
	if !ok || got != "10.0.0.42" {
		t.Fatalf("got (%q, %v), want 10.0.0.42", got, ok)
	}

	onlyLoop := []byte(`{"result":[{"name":"lo","ip-addresses":[{"ip-address":"127.0.0.1","ip-address-type":"ipv4","prefix":8}]}]}`)
	if ip, ok := proxmox.PickFirstNonLoopbackIPv4(onlyLoop); ok {
		t.Fatalf("loopback-only should fail, got %q", ip)
	}
}

func TestWaitGuestIPv4Success(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api2/json/access/ticket":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"ticket": "t", "CSRFPreventionToken": "c"},
			})
		case "/api2/json/nodes/pve/qemu/200/agent/network-get-interfaces":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"result": []map[string]any{
						{
							"name": "eth0",
							"ip-addresses": []map[string]any{
								{"ip-address": "192.168.1.50", "ip-address-type": "ipv4", "prefix": 24},
							},
						},
					},
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

	ip, err := client.WaitGuestIPv4(context.Background(), 200, time.Second, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("WaitGuestIPv4: %v", err)
	}
	if ip != "192.168.1.50" {
		t.Fatalf("ip = %q", ip)
	}
}
