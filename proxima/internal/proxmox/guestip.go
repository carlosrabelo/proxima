package proxmox

import (
	"encoding/json"
	"net"
	"strings"
)

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
	var payload agentInterfacePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		// Some Proxmox responses wrap under "data" / "result" differently;
		// also accept a bare result array via data envelope used by API.
		var wrapped struct {
			Data json.RawMessage `json:"data"`
		}
		if err2 := json.Unmarshal(raw, &wrapped); err2 != nil {
			return "", false
		}
		if err3 := json.Unmarshal(wrapped.Data, &payload); err3 != nil {
			var onlyResult agentInterfacePayload
			if err4 := json.Unmarshal(wrapped.Data, &onlyResult); err4 != nil {
				// data may itself be {"result":[...]}
				var inner struct {
					Result []agentIface `json:"result"`
				}
				if err5 := json.Unmarshal(wrapped.Data, &inner); err5 != nil {
					return "", false
				}
				payload.Result = inner.Result
			} else {
				payload = onlyResult
			}
		}
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
