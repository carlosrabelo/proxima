package config

type Config struct {
	Proxmox ProxmoxConfig `yaml:"proxmox"`
	SSH     SSHConfig     `yaml:"ssh"`
	VMs     []VMConfig    `yaml:"vms"`
}

type ProxmoxConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Node     string `yaml:"node"`
	VMIPBase string `yaml:"vm_ip_base"`
}

type SSHConfig struct {
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	KeyPath      string `yaml:"key_path"`
	Port         int    `yaml:"port"`
	CopyLocalKey bool   `yaml:"copy_local_key"`
}

type VMConfig struct {
	Name       string         `yaml:"name"`
	VMID       int            `yaml:"vmid"`
	Cores      int            `yaml:"cores"`
	Memory     int            `yaml:"memory"`
	DiskSize   string         `yaml:"disk_size"`
	Network    NetworkConfig  `yaml:"network"`
	OSTemplate string         `yaml:"os_template"`
	Tags       []string       `yaml:"tags"`
	AutoStart  bool           `yaml:"auto_start"`
	SSH        SSHVMConfig    `yaml:"ssh"`
	Scripts    []ScriptConfig `yaml:"scripts"`
}

type NetworkConfig struct {
	Bridge string `yaml:"bridge"`
	VLAN   int    `yaml:"vlan"`
	Model  string `yaml:"model"`
}

type SSHVMConfig struct {
	User           string   `yaml:"user"`
	Password       string   `yaml:"password"`
	KeyPath        string   `yaml:"key_path"`
	AuthorizedKeys []string `yaml:"authorized_keys"`
	CopyLocalKey   bool     `yaml:"copy_local_key"`
}

type ScriptConfig struct {
	Name    string   `yaml:"name"`
	Path    string   `yaml:"path"`
	Args    []string `yaml:"args"`
	Timeout int      `yaml:"timeout"`
}
