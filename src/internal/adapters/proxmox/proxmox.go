package proxmox

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"proxima/internal/core/domain"
	"proxima/internal/core/ports"
	"strings"
	"time"
)

type ProxmoxAdapter struct {
	client   *http.Client
	host     string
	port     int
	user     string
	password string
	node     string
	token    string
	apiToken string
	vmIPBase string
}

type ProxmoxVM struct {
	VMID    int     `json:"vmid"`
	Name    string  `json:"name"`
	Status  string  `json:"status"`
	CPU     float64 `json:"cpu"`
	Memory  int     `json:"mem"`
	MaxMem  int     `json:"maxmem"`
	Disk    int     `json:"disk"`
	MaxDisk int     `json:"maxdisk"`
	Uptime  int     `json:"uptime"`
}

type ProxmoxLoginResponse struct {
	Ticket              string `json:"ticket"`
	CSRFPreventionToken string `json:"CSRFPreventionToken"`
}

func NewProxmoxAdapter(host string, port int, user, password, node string) *ProxmoxAdapter {
	return NewProxmoxAdapterWithVMIPBase(host, port, user, password, node, "")
}

func NewProxmoxAdapterWithVMIPBase(host string, port int, user, password, node, vmIPBase string) *ProxmoxAdapter {
	return NewProxmoxAdapterWithToken(host, port, user, password, "", node, vmIPBase)
}

func NewProxmoxAdapterWithToken(host string, port int, user, password, apiToken, node, vmIPBase string) *ProxmoxAdapter {
	// Configure HTTP client with insecure TLS for self-signed certificates
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// If no VM IP base provided, extract from host (remove last octet)
	if vmIPBase == "" {
		parts := strings.Split(host, ".")
		if len(parts) == 4 {
			vmIPBase = strings.Join(parts[:3], ".")
		} else {
			vmIPBase = "192.168.1" // fallback
		}
	}

	return &ProxmoxAdapter{
		client:   &http.Client{Timeout: 30 * time.Second, Transport: tr},
		host:     host,
		port:     port,
		user:     user,
		password: password,
		node:     node,
		apiToken: apiToken,
		vmIPBase: vmIPBase,
	}
}

func (p *ProxmoxAdapter) login() error {
	// If API token is provided, use token authentication
	if p.apiToken != "" {
		log.Printf("Using API token authentication for Proxmox at %s:%d", p.host, p.port)
		p.token = p.apiToken
		return nil
	}

	// Otherwise use password authentication
	log.Printf("Logging into Proxmox at %s:%d as user %s", p.host, p.port, p.user)

	url := fmt.Sprintf("https://%s:%d/api2/json/access/ticket", p.host, p.port)

	data := map[string]string{
		"username": p.user,
		"password": p.password,
	}

	jsonData, _ := json.Marshal(data)

	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to login to Proxmox: %v", err)
		return fmt.Errorf("failed to login to Proxmox: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var loginResp struct {
		Data ProxmoxLoginResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	p.token = loginResp.Data.Ticket
	return nil
}

func (p *ProxmoxAdapter) setAuthHeader(req *http.Request) {
	// Use token authentication if API token is available
	if p.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", p.apiToken))
	} else {
		req.Header.Set("Cookie", fmt.Sprintf("PVEAuthCookie=%s", p.token))
	}
}

func (p *ProxmoxAdapter) Create(vm *domain.VM) error {
	log.Printf("Creating VM %s (ID: %d) with %d cores, %d MB memory", vm.Name, vm.ID, vm.Cores, vm.Memory)

	if p.token == "" {
		if err := p.login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu", p.host, p.port, p.node)

	// Build network configuration with VLAN support
	netConfig := fmt.Sprintf("model=%s,bridge=%s", vm.Network.Model, vm.Network.Bridge)
	if vm.Network.VLAN > 0 {
		netConfig += fmt.Sprintf(",tag=%d", vm.Network.VLAN)
	}

	data := map[string]any{
		"vmid":       vm.ID,
		"name":       vm.Name,
		"cores":      vm.Cores,
		"memory":     vm.Memory,
		"scsi0":      vm.DiskSize,
		"net0":       netConfig,
		"ostemplate": vm.OSTemplate,
	}

	jsonData, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	p.setAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create VM %s (ID: %d), status: %d, response: %s", vm.Name, vm.ID, resp.StatusCode, string(body))
	}

	return nil
}

func (p *ProxmoxAdapter) GetByID(id int) (*domain.VM, error) {
	if p.token == "" {
		if err := p.login(); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/status/current", p.host, p.port, p.node, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read VM response: %w", err)
	}

	var vmResp struct {
		Data ProxmoxVM `json:"data"`
	}

	if err := json.Unmarshal(body, &vmResp); err != nil {
		return nil, fmt.Errorf("failed to parse VM response: %w", err)
	}

	vm := domain.NewVM(vmResp.Data.Name, vmResp.Data.VMID)
	vm.Status = domain.VMStatus(vmResp.Data.Status)
	vm.Memory = vmResp.Data.Memory
	vm.Cores = int(vmResp.Data.CPU)

	return vm, nil
}

func (p *ProxmoxAdapter) GetByName(name string) (*domain.VM, error) {
	vms, err := p.List()
	if err != nil {
		return nil, err
	}

	for _, vm := range vms {
		if vm.Name == name {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("VM with name %s not found", name)
}

func (p *ProxmoxAdapter) Update(vm *domain.VM) error {
	return fmt.Errorf("update operation not implemented for Proxmox adapter")
}

func (p *ProxmoxAdapter) Delete(id int) error {
	if p.token == "" {
		if err := p.login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d", p.host, p.port, p.node, id)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete VM %d, status: %d, response: %s", id, resp.StatusCode, string(body))
	}

	return nil
}

func (p *ProxmoxAdapter) List() ([]*domain.VM, error) {
	if p.token == "" {
		if err := p.login(); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu", p.host, p.port, p.node)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed - check credentials in config.yaml (user: %s, host: %s)", p.user, p.host)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error - status: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read VMs response: %w", err)
	}

	var vmsResp struct {
		Data []ProxmoxVM `json:"data"`
	}

	if err := json.Unmarshal(body, &vmsResp); err != nil {
		return nil, fmt.Errorf("failed to parse VMs response: %w", err)
	}

	var vms []*domain.VM
	for _, vmData := range vmsResp.Data {
		vm := domain.NewVM(vmData.Name, vmData.VMID)
		vm.Status = domain.VMStatus(vmData.Status)
		vm.Memory = vmData.Memory
		vm.Cores = int(vmData.CPU)
		vms = append(vms, vm)
	}

	return vms, nil
}

func (p *ProxmoxAdapter) Start(id int) error {
	if p.token == "" {
		if err := p.login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/status/start", p.host, p.port, p.node, id)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to start VM %d, status: %d, response: %s", id, resp.StatusCode, string(body))
	}

	return nil
}

func (p *ProxmoxAdapter) Stop(id int) error {
	if p.token == "" {
		if err := p.login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/status/stop", p.host, p.port, p.node, id)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to stop VM %d, status: %d, response: %s", id, resp.StatusCode, string(body))
	}

	return nil
}

func (p *ProxmoxAdapter) Shutdown(id int) error {
	if p.token == "" {
		if err := p.login(); err != nil {
			return err
		}
	}

	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/status/shutdown", p.host, p.port, p.node, id)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to shutdown VM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to shutdown VM %d, status: %d, response: %s", id, resp.StatusCode, string(body))
	}

	return nil
}

func (p *ProxmoxAdapter) GetStatus(id int) (domain.VMStatus, error) {
	vm, err := p.GetByID(id)
	if err != nil {
		return "", err
	}
	return vm.Status, nil
}

func (p *ProxmoxAdapter) GetVMIP(id int) (string, error) {
	if p.token == "" {
		if err := p.login(); err != nil {
			return "", err
		}
	}

	// Try to get IP from QEMU Agent first
	url := fmt.Sprintf("https://%s:%d/api2/json/nodes/%s/qemu/%d/agent/network-get-interfaces", p.host, p.port, p.node, id)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get network interfaces: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read network response: %w", err)
		}

		var ifacesResp struct {
			Data []struct {
				Name        string `json:"name"`
				IPAddresses []struct {
					IPAddress string `json:"ip-address"`
					Type      string `json:"type"`
				} `json:"ip-addresses"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &ifacesResp); err == nil {
			for _, iface := range ifacesResp.Data {
				if iface.Name == "eth0" || iface.Name == "ens18" {
					for _, ip := range iface.IPAddresses {
						if ip.Type == "ipv4" && ip.IPAddress != "" {
							return ip.IPAddress, nil
						}
					}
				}
			}
		}
	}

	// Generate fallback IP based on VMID using configurable base
	return fmt.Sprintf("%s.%d", p.vmIPBase, id+100), nil
}

var _ ports.VMRepository = (*ProxmoxAdapter)(nil)
