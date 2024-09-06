package cli_test

import (
	"bytes"
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
)

func TestProxmoxPingSuccess(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api2/json/access/ticket" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{
					"ticket":              "PVE:ticket",
					"CSRFPreventionToken": "csrf",
				},
			})
		case r.URL.Path == "/api2/json/version" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]string{"version": "8.2.2"},
			})
		case r.URL.Path == "/api2/json/nodes/pve/status" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	host, port := pingHostPort(t, srv.URL)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "proxima.yaml")
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
`, host, port)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"proxmox", "ping", "-c", cfgPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr=%q", code, stderr.String())
	}
	want := "proxmox: ok\nversion: 8.2.2\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestProxmoxPingConfigErrorExitOne(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("proxmox_host: \"\"\nnode: pve\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"proxmox", "ping", "-c", path}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "Error: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestProxmoxPingAPIErrorExitThree(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	host, port := pingHostPort(t, srv.URL)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "proxima.yaml")
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
`, host, port)
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := cli.ExecuteWithArgs([]string{"proxmox", "ping", "-c", cfgPath}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.HasPrefix(stderr.String(), "Error: ") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "proxmox authentication failed") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func pingHostPort(t *testing.T, raw string) (string, int) {
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
