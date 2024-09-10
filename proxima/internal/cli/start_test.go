package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"proxima/proxima/internal/cli"
	"proxima/proxima/internal/config"
	"proxima/proxima/internal/proxmox"
)

func TestStartCommandPrintsIP(t *testing.T) {
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
								{"ip-address": "10.10.10.10", "ip-address-type": "ipv4", "prefix": 24},
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
agent_wait_timeout: "2s"
agent_poll_interval: "50ms"
`, host, port)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cli.SetStartClientFactoryForTest(func(cfg config.Config) (*proxmox.Client, error) {
		c, err := proxmox.NewClient(cfg)
		if err != nil {
			return nil, err
		}
		c.SetSSHOpenForTest(func(ctx context.Context) (proxmox.SSHSession, error) {
			return &startFakeSSH{}, nil
		})
		return c, nil
	})
	t.Cleanup(cli.ResetStartClientFactoryForTest)

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"start", "-c", cfgPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	if stdout.String() != "vmid=200 ip=10.10.10.10\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "==> Start: done (vmid=200)") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "==> WaitIP: done (10.10.10.10)") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

type startFakeSSH struct{}

func (s *startFakeSSH) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	_ = ctx
	if strings.HasPrefix(command, "qm status ") {
		return []byte("status: stopped\n"), nil
	}
	return nil, nil
}

func (s *startFakeSSH) Close() error { return nil }

func startHostPort(t *testing.T, raw string) (string, int) {
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
