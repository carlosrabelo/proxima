package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func TestIPCommandTimeoutExitThree(t *testing.T) {
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
							"name": "lo",
							"ip-addresses": []map[string]any{
								{"ip-address": "127.0.0.1", "ip-address-type": "ipv4", "prefix": 8},
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

	host, port := startHostPort(t, srv.URL)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "ok.yaml")
	content := fmt.Sprintf(`
proxmox_host: %q
proxmox_port: %d
node: "pve"
user: "root@pam"
password: "secret"
vmid: 200
name: "alpine-docker"
memory: 2048
cores: 2
disk_size: "8G"
root_password: "change-me"
insecure_tls: true
agent_wait_timeout: "80ms"
agent_poll_interval: "20ms"
`, host, port)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.SetIPClientFactoryForTest(func(cfg config.Config) (*proxmox.Client, error) {
		return proxmox.NewClient(cfg)
	})
	t.Cleanup(cli.ResetIPClientFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"ip", "-c", cfgPath}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error: guest ip wait timed out after") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> WaitIP: starting") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
