package proxmox_test

import (
	"testing"

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
