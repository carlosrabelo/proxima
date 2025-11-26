# Proxima

A modern CLI tool for Proxmox VM management with clean architecture and simplified command structure.

## Features

- **Simplified Command Structure**: `proxima <host> <command>` and `proxima <yaml> <command>`
- **Positional Arguments**: No more complex flags for VMID - use `proxima <host> start <vmid>`
- **Graceful Shutdown**: Both `stop` (immediate) and `shutdown` (graceful) commands
- **Multiple Operation Modes**: Config file, SSH direct, and interactive login
- **Idempotent Infrastructure**: VM creation with existence verification
- **Clean Architecture**: Hexagonal architecture with proper separation of concerns
- **Zero-Touch Installation**: Silent installation without prompts
- **No Help Flags**: Clean interface without `-h`/`--help` flags

## Command Structure

### Basic Usage
```bash
proxima <host> <command> [arguments]
proxima <yaml> <command> [arguments]
proxima help <command>
```

### Operation Modes

#### 1. Host Mode (Direct SSH)
```bash
proxima 10.13.250.11 list
proxima 10.13.250.11 start 100
proxima 10.13.250.11 shutdown 100
proxima 10.13.250.11 stop 100
proxima 10.13.250.11 delete 100
```

#### 2. Config File Mode
```bash
proxima config.yaml create --name web-server
proxima config.yaml
```

#### 3. Interactive Login Mode
```bash
proxima --login 10.13.250.11 list
```

## Available Commands

| Command | Description | Example | Arguments |
|---------|-------------|---------|-----------|
| `list` | List all VMs | `proxima 10.13.250.11 list` | None |
| `start <vmid>` | Start a VM | `proxima 10.13.250.11 start 100` | VM ID (positional) |
| `stop <vmid>` | Stop a VM immediately | `proxima 10.13.250.11 stop 100` | VM ID (positional) |
| `shutdown <vmid>` | Shutdown a VM gracefully | `proxima 10.13.250.11 shutdown 100` | VM ID (positional) |
| `delete <vmid>` | Delete a VM | `proxima 10.13.250.11 delete 100` | VM ID (positional) |
| `create --name <vm_name>` | Create VM from config | `proxima config.yaml create --name web` | VM name (flag) |

### Help System
```bash
proxima                    # Show usage overview
proxima help <command>     # Show specific command help
```

## Configuration

### Config File Structure (`config.yaml`)

```yaml
proxmox:
  host: "192.168.1.100"        # Proxmox server IP
  port: 8006                   # API port
  user: "root@pam"             # Username
  password: "password"          # Password
  node: "pve"                  # Node name
  vm_ip_base: "192.168.1"      # Base IP for VMs

ssh:
  user: "root"                 # Default SSH user
  password: ""                 # SSH password (optional)
  key_path: ""                 # SSH key path (optional)
  port: 22                     # SSH port
  copy_local_key: false        # Copy local key to VMs

vms:
  - name: "web-server"         # VM name
    vmid: 100                  # VM ID
    cores: 2                   # CPU cores
    memory: 2048               # Memory in MB
    disk_size: "20G"           # Disk size
    network:
      bridge: "vmbr0"          # Network bridge
      vlan: 100                # VLAN ID
      model: "virtio"          # Network model
    os_template: "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
    ssh:
      authorized_keys:          # Pre-authorized SSH keys
        - "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."
```

## Installation

### Build
```bash
make build
```

### Install System-wide
```bash
make install
```
- Installs to `$HOME/.local/bin`
- Silent installation (no prompts)
- Automatic PATH detection

### Uninstall
```bash
make uninstall
```

### Clean Build
```bash
make clean
```

## Architecture

```
src/
├── cmd/                    # Main CLI application
├── internal/
│   ├── core/
│   │   ├── domain/        # Core entities (VM, Command, etc.)
│   │   ├── ports/         # Interface definitions
│   │   └── service/       # Business logic orchestration
│   └── adapters/
│       ├── config/        # YAML configuration management
│       ├── proxmox/       # Proxmox API integration
│       ├── proxmox_ssh/   # Direct SSH integration
│       └── ssh/           # SSH client and key management
└── config.yaml           # Configuration file
```

### Design Patterns

- **Hexagonal Architecture**: Clean separation between domain and infrastructure
- **Repository Pattern**: Abstraction over different Proxmox access methods
- **Service Layer**: Business logic orchestration
- **Adapter Pattern**: Multiple integration methods (API, SSH)

## Authentication Methods

### SSH Direct Mode
- Uses `~/.ssh/id_rsa` by default
- Supports multiple key formats (RSA, Ed25519, ECDSA)
- No password prompts
- Fastest and most secure method

### Interactive Login Mode
- Prompts for SSH username and password
- Uses `sshpass` for password authentication
- Useful when keys are not configured

### Config File Mode
- Uses Proxmox API with credentials from config.yaml
- Supports both password and token authentication
- Handles self-signed certificates

## Advanced Features

### Idempotent VM Creation
```bash
proxima config.yaml
```
Output:
```
Processing 3 VM(s) from config.yaml...

[OK] VM 'web-server' (ID: 101) already exists
[CREATE] Creating VM 'database' (ID: 102)...
[OK] VM 'database' (ID: 102) created successfully
[CREATE] Creating VM 'cache' (ID: 103)...
[OK] VM 'cache' (ID: 103) created successfully

[INFO] Current VM status:
VMID   Name                 Status     CPU      Memory  
----------------------------------------------------
101    web-server           running    2        2048 MB
102    database             stopped    4        4096 MB
103    cache                stopped    2        1024 MB
```

### Graceful vs Immediate Shutdown
```bash
# Graceful shutdown (waits for VM to properly shutdown)
proxima 10.13.250.11 shutdown 100

# Immediate shutdown (force stop)
proxima 10.13.250.11 stop 100
```

The `shutdown` command:
- Waits for VM to be fully running if starting
- Sends graceful shutdown signal
- Waits up to 60 seconds for VM to stop
- Provides clear feedback during the process

The `stop` command:
- Immediately stops the VM
- No waiting period
- Force termination

### Dynamic IP Resolution
- **Primary**: QEMU Agent for real IP discovery
- **Fallback**: Configurable base IP pattern (`vm_ip_base.{vmid+100}`)
- No hardcoded IP addresses

### SSH Key Management
- Automatic key copying to VMs
- Support for multiple authentication methods
- Fallback key discovery (`id_ed25519`, `id_rsa`, etc.)

## Dependencies

- `cobra`: CLI framework
- `yaml`: YAML parser
- `golang.org/x/crypto/ssh`: SSH client
- `crypto/tls`: TLS configuration for HTTPS

## Quality & Reliability

- **Configuration Validation**: Comprehensive config validation with detailed error messages
- **Error Handling**: Structured error messages with context and suggestions
- **Logging**: Detailed logging for debugging and monitoring
- **Testing**: Unit tests for core components and adapters

## Getting Help

```bash
proxima                    # Show usage overview
proxima help list         # Show specific command help
proxima help start        # Show start command help
proxima help shutdown     # Show shutdown command help
```

The help system provides:
- Usage overview with all available commands
- Specific command help with syntax and examples
- Clear distinction between stop (immediate) and shutdown (graceful)
- No confusing `-h`/`--help` flags

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

[Add your license here]