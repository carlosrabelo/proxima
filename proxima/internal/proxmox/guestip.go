package proxmox

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// WaitGuestIPv4 polls the guest agent until a non-loopback IPv4 appears or timeout elapses.
func (c *Client) WaitGuestIPv4(ctx context.Context, vmid int, timeout, interval time.Duration) (string, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		raw, err := c.fetchGuestNetwork(ctx, vmid)
		if err == nil {
			if ip, ok := PickFirstNonLoopbackIPv4(raw); ok {
				return ip, nil
			}
		}
		if time.Now().After(deadline) || time.Now().Equal(deadline) {
			return "", fmt.Errorf("guest ip wait timed out after %s", timeout)
		}
		remaining := time.Until(deadline)
		sleep := interval
		if sleep > remaining {
			sleep = remaining
		}
		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("guest ip wait timed out after %s", timeout)
		}
	}
}

func (c *Client) fetchGuestNetwork(ctx context.Context, vmid int) ([]byte, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/agent/network-get-interfaces", url.PathEscape(c.node), vmid)
	body, err := c.get(ctx, path)
	if err == nil {
		return body, nil
	}
	sess, sshErr := c.SSH(ctx)
	if sshErr != nil {
		return nil, err
	}
	defer sess.Close()
	out, sshErr := sess.CombinedOutput(ctx, fmt.Sprintf("qm guest cmd %d network-get-interfaces", vmid))
	if sshErr != nil {
		return nil, err
	}
	return out, nil
}

// agentInterfacePayload is a subset of QEMU guest agent network-get-interfaces.
type agentInterfacePayload struct {
	Result []agentIface `json:"result"`
}

type agentIface struct {
	Name            string        `json:"name"`
	IPAddresses     []agentIPAddr `json:"ip-addresses"`
	HardwareAddress string        `json:"hardware-address"`
}

type agentIPAddr struct {
	IPAddress     string `json:"ip-address"`
	IPAddressType string `json:"ip-address-type"`
	Prefix        int    `json:"prefix"`
}

// PickFirstNonLoopbackIPv4 returns the first non-loopback IPv4 from guest-agent JSON.
// Prefers interfaces whose name is not "lo".
func PickFirstNonLoopbackIPv4(raw []byte) (string, bool) {
	payload, ok := parseAgentInterfaces(raw)
	if !ok {
		return "", false
	}

	var fallback string
	for _, iface := range payload.Result {
		ip := firstIPv4(iface)
		if ip == "" {
			continue
		}
		if iface.Name == "lo" || strings.HasPrefix(iface.Name, "lo.") {
			if fallback == "" {
				fallback = ip
			}
			continue
		}
		return ip, true
	}
	if fallback != "" {
		return fallback, true
	}
	return "", false
}

func parseAgentInterfaces(raw []byte) (agentInterfacePayload, bool) {
	var payload agentInterfacePayload
	if err := json.Unmarshal(raw, &payload); err == nil && len(payload.Result) > 0 {
		return payload, true
	}
	var wrapped struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil || len(wrapped.Data) == 0 {
		return agentInterfacePayload{}, len(payload.Result) > 0
	}
	if err := json.Unmarshal(wrapped.Data, &payload); err == nil && len(payload.Result) > 0 {
		return payload, true
	}
	return agentInterfacePayload{}, false
}

func firstIPv4(iface agentIface) string {
	for _, addr := range iface.IPAddresses {
		if addr.IPAddressType != "" && !strings.EqualFold(addr.IPAddressType, "ipv4") {
			continue
		}
		ip := net.ParseIP(addr.IPAddress)
		if ip == nil {
			continue
		}
		ip4 := ip.To4()
		if ip4 == nil {
			continue
		}
		if ip4[0] == 127 {
			continue
		}
		return ip4.String()
	}
	return ""
}
