package ssh

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"proxima/internal/core/domain"
	"proxima/internal/core/ports"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHAdapter struct {
	config      SSHConfig
	proxmoxRepo ports.VMRepository
}

type SSHConfig struct {
	User         string
	Password     string
	KeyPath      string
	Port         int
	CopyLocalKey bool
}

func NewSSHAdapter(user, password, keyPath string, port int) *SSHAdapter {
	return &SSHAdapter{
		config: SSHConfig{
			User:         user,
			Password:     password,
			KeyPath:      keyPath,
			Port:         port,
			CopyLocalKey: false,
		},
	}
}

func NewSSHAdapterWithKeyCopy(user, password, keyPath string, port int, copyLocalKey bool) *SSHAdapter {
	return &SSHAdapter{
		config: SSHConfig{
			User:         user,
			Password:     password,
			KeyPath:      keyPath,
			Port:         port,
			CopyLocalKey: copyLocalKey,
		},
	}
}

func NewSSHAdapterWithProxmox(user, password, keyPath string, port int, copyLocalKey bool, proxmoxRepo ports.VMRepository) *SSHAdapter {
	return &SSHAdapter{
		config: SSHConfig{
			User:         user,
			Password:     password,
			KeyPath:      keyPath,
			Port:         port,
			CopyLocalKey: copyLocalKey,
		},
		proxmoxRepo: proxmoxRepo,
	}
}

func (s *SSHAdapter) getSSHClient(host string) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Try explicit key if configured
	if s.config.KeyPath != "" {
		key, err := os.ReadFile(s.config.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key from %s: %w", s.config.KeyPath, err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key from %s: %w", s.config.KeyPath, err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		// Fallback to default SSH keys
		if err := s.tryDefaultSSHKeys(&authMethods); err != nil {
			return nil, fmt.Errorf("failed to load default SSH keys from ~/.ssh/: %w", err)
		}
	}

	if s.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(s.config.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH authentication method available - please configure password, key_path, or ensure default SSH keys exist in ~/.ssh/")
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, s.config.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH at %s:%d with user %s: %w", host, s.config.Port, s.config.User, err)
	}

	return client, nil
}

func (s *SSHAdapter) ExecuteCommand(vmid int, command string, args []string, timeout int) (*domain.Command, error) {
	host, err := s.getVMHost(vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM host: %w", err)
	}

	cmd := domain.NewCommand(vmid, command, args, timeout)
	cmd.Start()

	client, err := s.getSSHClient(host)
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, err
	}
	defer client.Close()

	fullCommand := command
	for _, arg := range args {
		fullCommand += " " + arg
	}

	session, err := client.NewSession()
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(fullCommand)
	if err != nil {
		outputStr := string(output)
		cmd.Fail(outputStr + " - " + err.Error())
		return cmd, fmt.Errorf("command '%s' failed on VM %d: output: %s, error: %w", fullCommand, vmid, outputStr, err)
	}

	cmd.Complete(string(output))
	return cmd, nil
}

func (s *SSHAdapter) ExecuteScript(vmid int, scriptPath string, args []string, timeout int) (*domain.Command, error) {
	host, err := s.getVMHost(vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM host: %w", err)
	}

	cmd := domain.NewCommand(vmid, "bash", append([]string{scriptPath}, args...), timeout)
	cmd.Start()

	client, err := s.getSSHClient(host)
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, fmt.Errorf("failed to read script file %s: %w", scriptPath, err)
	}

	scriptName := filepath.Base(scriptPath)

	err = s.copyScriptToRemote(client, scriptName, scriptContent)
	if err != nil {
		cmd.Fail(err.Error())
		return cmd, fmt.Errorf("failed to copy script to remote: %w", err)
	}
	defer s.cleanupRemoteScript(client, scriptName)

	fullCommand := fmt.Sprintf("chmod +x /tmp/%s && /tmp/%s", scriptName, scriptName)
	for _, arg := range args {
		fullCommand += " " + arg
	}

	output, err := session.CombinedOutput(fullCommand)
	if err != nil {
		outputStr := string(output)
		cmd.Fail(outputStr + " - " + err.Error())
		return cmd, fmt.Errorf("script '%s' execution failed on VM %d: output: %s, error: %w", scriptName, vmid, outputStr, err)
	}

	cmd.Complete(string(output))
	return cmd, nil
}

func (s *SSHAdapter) copyScriptToRemote(client *ssh.Client, scriptName string, content []byte) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(content)
	}()

	err = session.Run(fmt.Sprintf("cat > /tmp/%s", scriptName))
	if err != nil {
		return err
	}

	return nil
}

func (s *SSHAdapter) cleanupRemoteScript(client *ssh.Client, scriptName string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run(fmt.Sprintf("rm -f /tmp/%s", scriptName))
}

func (s *SSHAdapter) tryDefaultSSHKeys(authMethods *[]ssh.AuthMethod) error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	sshDir := filepath.Join(currentUser.HomeDir, ".ssh")

	// Priority order of default keys
	defaultKeys := []string{
		"id_ed25519",
		"id_rsa",
		"id_ecdsa",
		"id_dsa",
	}

	for _, keyName := range defaultKeys {
		keyPath := filepath.Join(sshDir, keyName)
		if _, err := os.Stat(keyPath); err == nil {
			key, err := os.ReadFile(keyPath)
			if err != nil {
				continue // Tenta próxima chave
			}

			signer, err := ssh.ParsePrivateKey(key)
			if err != nil {
				continue // Try next key
			}

			*authMethods = append(*authMethods, ssh.PublicKeys(signer))
			return nil // Success, use first key found
		}
	}

	return fmt.Errorf("no default SSH keys found in %s", sshDir)
}

func (s *SSHAdapter) CopyLocalPublicKey(vmid int) error {
	if !s.config.CopyLocalKey {
		return fmt.Errorf("key copy is disabled")
	}

	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	publicKeyPath := filepath.Join(currentUser.HomeDir, ".ssh", "id_rsa.pub")
	publicKey, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key from %s: %w", publicKeyPath, err)
	}

	host, err := s.getVMHost(vmid)
	if err != nil {
		return fmt.Errorf("failed to get VM host: %w", err)
	}
	client, err := s.getSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Create .ssh directory if it doesn't exist and add the key
	command := fmt.Sprintf("mkdir -p ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", string(publicKey))
	output, err := session.CombinedOutput(command)
	if err != nil {
		return fmt.Errorf("failed to copy public key to VM %d: %s - %w", vmid, string(output), err)
	}

	return nil
}

func (s *SSHAdapter) getVMHost(vmid int) (string, error) {
	// Get real IP from Proxmox - this is the only supported method
	if s.proxmoxRepo == nil {
		return "", fmt.Errorf("no Proxmox repository available to determine VM IP for VM %d", vmid)
	}

	proxmoxAdapter, ok := s.proxmoxRepo.(interface{ GetVMIP(int) (string, error) })
	if !ok {
		return "", fmt.Errorf("Proxmox repository does not support IP resolution for VM %d", vmid)
	}

	ip, err := proxmoxAdapter.GetVMIP(vmid)
	if err != nil {
		return "", fmt.Errorf("failed to get IP for VM %d: %w", vmid, err)
	}

	if ip == "" {
		return "", fmt.Errorf("empty IP returned for VM %d", vmid)
	}

	return ip, nil
}

func (s *SSHAdapter) GetCommandHistory(vmid int) ([]*domain.Command, error) {
	return []*domain.Command{}, fmt.Errorf("command history not implemented for SSH adapter")
}

var _ ports.SSHRepository = (*SSHAdapter)(nil)
