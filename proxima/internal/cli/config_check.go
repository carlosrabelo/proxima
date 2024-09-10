package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"proxima/proxima/internal/config"
)

type configFlags struct {
	configPath        string
	proxmoxHost       string
	proxmoxPort       int
	sshPort           int
	node              string
	user              string
	password          string
	tokenID           string
	tokenSecret       string
	vmid              int
	name              string
	memory            int
	cores             int
	diskSize          string
	rootPassword      string
	storage           string
	bridge            string
	output            string
	workDir           string
	insecureTLS       bool
	sshUser           string
	sshPrivateKeyPath string
	agentTimeout      time.Duration
	agentPollInterval time.Duration
}

func newConfigCheckCommand() *cobra.Command {
	f := &configFlags{}
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Load, validate, and print redacted configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigCheck(cmd, f)
		},
	}

	bindConfigFlags(cmd, f)
	return cmd
}

func bindConfigFlags(cmd *cobra.Command, f *configFlags) {
	cmd.Flags().StringVarP(&f.configPath, "config", "c", "", "config file path (.yaml, .yml, or .json)")
	cmd.Flags().StringVar(&f.proxmoxHost, "proxmox-host", "", "Proxmox host")
	cmd.Flags().IntVar(&f.proxmoxPort, "proxmox-port", 0, "Proxmox API port")
	cmd.Flags().IntVar(&f.sshPort, "ssh-port", 0, "SSH port")
	cmd.Flags().StringVar(&f.node, "node", "", "Proxmox node name")
	cmd.Flags().StringVar(&f.user, "user", "", "password-auth user")
	cmd.Flags().StringVar(&f.password, "password", "", "password-auth password")
	cmd.Flags().StringVar(&f.tokenID, "token-id", "", "API token id")
	cmd.Flags().StringVar(&f.tokenSecret, "token-secret", "", "API token secret")
	cmd.Flags().IntVar(&f.vmid, "vmid", 0, "VM id")
	cmd.Flags().StringVar(&f.name, "name", "", "VM name")
	cmd.Flags().IntVar(&f.memory, "memory", 0, "memory MiB")
	cmd.Flags().IntVar(&f.cores, "cores", 0, "CPU cores")
	cmd.Flags().StringVar(&f.diskSize, "disk-size", "", "disk size")
	cmd.Flags().StringVar(&f.rootPassword, "root-password", "", "guest root password")
	cmd.Flags().StringVar(&f.storage, "storage", "", "Proxmox storage")
	cmd.Flags().StringVar(&f.bridge, "bridge", "", "network bridge")
	cmd.Flags().StringVar(&f.output, "output", "", "local qcow2 output path")
	cmd.Flags().StringVar(&f.workDir, "work-dir", "", "work directory")
	cmd.Flags().BoolVar(&f.insecureTLS, "insecure-tls", false, "skip TLS verify")
	cmd.Flags().StringVar(&f.sshUser, "ssh-user", "", "SSH username")
	cmd.Flags().StringVar(&f.sshPrivateKeyPath, "ssh-private-key", "", "SSH private key path")
	cmd.Flags().DurationVar(&f.agentTimeout, "agent-timeout", 0, "guest agent wait timeout")
	cmd.Flags().DurationVar(&f.agentPollInterval, "agent-poll-interval", 0, "guest agent poll interval")
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration helpers",
	}
	cmd.AddCommand(newConfigCheckCommand())
	return cmd
}

func runConfigCheck(cmd *cobra.Command, f *configFlags) error {
	if f.configPath == "" {
		return fmt.Errorf("config file must end in .yaml, .yml, or .json")
	}

	cfg, err := config.Load(f.configPath, overrideFromFlags(cmd, f))
	if err != nil {
		return err
	}
	if err := config.Validate(cfg); err != nil {
		return err
	}

	out, err := yaml.Marshal(redactedYAML(config.Redact(cfg)))
	if err != nil {
		return err
	}
	_, err = cmd.OutOrStdout().Write(out)
	return err
}

func overrideFromFlags(cmd *cobra.Command, f *configFlags) config.Override {
	var o config.Override
	fs := cmd.Flags()
	if fs.Changed("proxmox-host") {
		o.ProxmoxHost = &f.proxmoxHost
	}
	if fs.Changed("proxmox-port") {
		o.ProxmoxPort = &f.proxmoxPort
	}
	if fs.Changed("ssh-port") {
		o.SSHPort = &f.sshPort
	}
	if fs.Changed("node") {
		o.Node = &f.node
	}
	if fs.Changed("user") {
		o.User = &f.user
	}
	if fs.Changed("password") {
		o.Password = &f.password
	}
	if fs.Changed("token-id") {
		o.TokenID = &f.tokenID
	}
	if fs.Changed("token-secret") {
		o.TokenSecret = &f.tokenSecret
	}
	if fs.Changed("vmid") {
		o.VMID = &f.vmid
	}
	if fs.Changed("name") {
		o.VMName = &f.name
	}
	if fs.Changed("memory") {
		o.MemoryMiB = &f.memory
	}
	if fs.Changed("cores") {
		o.Cores = &f.cores
	}
	if fs.Changed("disk-size") {
		o.DiskSize = &f.diskSize
	}
	if fs.Changed("root-password") {
		o.RootPassword = &f.rootPassword
	}
	if fs.Changed("storage") {
		o.Storage = &f.storage
	}
	if fs.Changed("bridge") {
		o.Bridge = &f.bridge
	}
	if fs.Changed("output") {
		o.OutputQcow2 = &f.output
	}
	if fs.Changed("work-dir") {
		o.WorkDir = &f.workDir
	}
	if fs.Changed("insecure-tls") {
		o.InsecureTLS = &f.insecureTLS
	}
	if fs.Changed("ssh-user") {
		o.SSHUser = &f.sshUser
	}
	if fs.Changed("ssh-private-key") {
		o.SSHPrivateKeyPath = &f.sshPrivateKeyPath
	}
	if fs.Changed("agent-timeout") {
		o.AgentWaitTimeout = &f.agentTimeout
	}
	if fs.Changed("agent-poll-interval") {
		o.AgentPollInterval = &f.agentPollInterval
	}
	return o
}

type redactedView struct {
	ProxmoxHost       string `yaml:"proxmox_host"`
	ProxmoxPort       int    `yaml:"proxmox_port"`
	SSHPort           int    `yaml:"ssh_port"`
	Node              string `yaml:"node"`
	User              string `yaml:"user,omitempty"`
	Password          string `yaml:"password,omitempty"`
	TokenID           string `yaml:"token_id,omitempty"`
	TokenSecret       string `yaml:"token_secret,omitempty"`
	VMID              int    `yaml:"vmid"`
	Name              string `yaml:"name"`
	Memory            int    `yaml:"memory"`
	Cores             int    `yaml:"cores"`
	DiskSize          string `yaml:"disk_size"`
	RootPassword      string `yaml:"root_password"`
	Storage           string `yaml:"storage"`
	Bridge            string `yaml:"bridge"`
	Output            string `yaml:"output"`
	WorkDir           string `yaml:"work_dir"`
	InsecureTLS       bool   `yaml:"insecure_tls"`
	SSHUser           string `yaml:"ssh_user,omitempty"`
	SSHPrivateKeyPath string `yaml:"ssh_private_key_path,omitempty"`
}

func redactedYAML(c config.Config) redactedView {
	return redactedView{
		ProxmoxHost:       c.ProxmoxHost,
		ProxmoxPort:       c.ProxmoxPort,
		SSHPort:           c.SSHPort,
		Node:              c.Node,
		User:              c.Auth.User,
		Password:          c.Auth.Password,
		TokenID:           c.Auth.TokenID,
		TokenSecret:       c.Auth.TokenSecret,
		VMID:              c.VMID,
		Name:              c.VMName,
		Memory:            c.MemoryMiB,
		Cores:             c.Cores,
		DiskSize:          c.DiskSize,
		RootPassword:      c.RootPassword,
		Storage:           c.Storage,
		Bridge:            c.Bridge,
		Output:            c.OutputQcow2,
		WorkDir:           c.WorkDir,
		InsecureTLS:       c.InsecureTLS,
		SSHUser:           c.SSHUser,
		SSHPrivateKeyPath: c.SSHPrivateKeyPath,
	}
}
