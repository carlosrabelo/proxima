# Proxima User Guide

Proxima is a modern CLI tool for managing Proxmox VMs with simplified command structure and comprehensive features.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Command Structure](#command-structure)
- [Operation Modes](#operation-modes)
- [Available Commands](#available-commands)
- [Authentication Methods](#authentication-methods)
- [Network Configuration](#network-configuration)
- [Graceful vs Immediate Shutdown](#graceful-vs-immediate-shutdown)
- [Troubleshooting](#troubleshooting)

## Installation

### 1. Clone Repository
```bash
git clone <repository-url>
cd proxima
```

### 2. Build Binary
```bash
make build
```

### 3. Install System-wide
```bash
make install
```

### 4. Verify Installation
```bash
proxima
```

## Configuration

Proxima uses a YAML configuration file (`config.yaml`) to define Proxmox connection settings and VM configurations.

### Basic Configuration Structure

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

### Configuration Fields

#### Proxmox Section
- `host`: Proxmox server IP/hostname
- `port`: Proxmox API port (default: 8006)
- `user`: Proxmox username (e.g., "root@pam")
- `password`: Proxmox password
- `node`: Proxmox node name
- `vm_ip_base`: Base IP for VM IP generation

#### SSH Section
- `user`: Default SSH user
- `password`: Default SSH password
- `key_path`: Path to SSH private key (optional)
- `port`: SSH port (default: 22)
- `copy_local_key`: Copy local SSH key to VMs

#### VM Configuration
- `name`: VM name (required)
- `vmid`: Unique VM ID (required)
- `cores`: Number of CPU cores (required)
- `memory`: Memory in MB (required)
- `disk_size`: Disk size (required)
- `network`: Network configuration
- `os_template`: OS template path (required)
- `tags`: List of tags for organization
- `auto_start`: Auto-start VM after creation
- `ssh`: VM-specific SSH configuration
- `scripts`: Scripts to execute after VM creation

## Command Structure

Proxima uses a simplified command structure:

```bash
proxima <host> <command> [arguments]
proxima <yaml> <command> [arguments]
proxima help <command>
```

### Key Changes
- **Positional Arguments**: VMID is now positional, not a flag
- **No Help Flags**: No `-h`/`--help` flags, use `proxima help <command>`
- **Simplified Syntax**: `proxima <host> start 100` instead of `proxima <host> start --vmid 100`

## Operation Modes

### 1. Host Mode (Direct SSH)
```bash
proxima 10.13.250.11 list
proxima 10.13.250.11 start 100
proxima 10.13.250.11 shutdown 100
proxima 10.13.250.11 stop 100
proxima 10.13.250.11 delete 100
```
- Direct SSH connection using `~/.ssh/id_rsa`
- No password prompts
- Fastest method for day-to-day operations
- Uses `qm` commands via SSH

### 2. Config File Mode
```bash
proxima config.yaml create --name web-server
proxima config.yaml
```
- Uses YAML configuration file
- Idempotent VM creation (only creates if not exists)
- Infrastructure as Code approach

### 3. Interactive Login Mode
```bash
proxima --login 10.13.250.11 list
```
- Prompts for SSH username and password
- Useful when SSH keys are not configured
- Fallback authentication method

## Available Commands

### Basic Commands

#### List VMs
```bash
# Host mode
proxima 10.13.250.11 list

# Config file mode
proxima config.yaml

# Interactive mode
proxima --login 10.13.250.11 list
```

#### VM Lifecycle
```bash
# Start VM (positional VMID)
proxima 10.13.250.11 start 100

# Graceful shutdown VM (positional VMID)
proxima 10.13.250.11 shutdown 100

# Immediate stop VM (positional VMID)
proxima 10.13.250.11 stop 100

# Delete VM (positional VMID)
proxima 10.13.250.11 delete 100

# Create VM from config
proxima config.yaml create --name web-server
```

### Help System
```bash
# Show usage overview
proxima

# Show specific command help
proxima help list
proxima help start
proxima help shutdown
proxima help stop
proxima help delete
proxima help create
```

### Command Options

- `--login`: Use interactive login instead of SSH keys
- `--name <name>`: VM name for creation (flag)
- VMID is now positional, not a flag

## Authentication Methods

### SSH Direct Mode
- Uses `~/.ssh/id_rsa` by default
- Supports multiple key formats (RSA, Ed25519, ECDSA)
- No password prompts
- Fastest and most secure method

### Interactive Login Mode
- Prompts for SSH username and password
- Uses `sshpass` for password authentication
- Useful when SSH keys are not configured

### Config File Mode
- Uses Proxmox API with credentials from config.yaml
- Supports both password and token authentication
- Handles self-signed certificates

## Network Configuration

### VLAN Support
```yaml
network:
  bridge: "vmbr0"          # Network bridge
  vlan: 100                # VLAN ID (optional)
  model: "virtio"          # Network model
```

### Dynamic IP Resolution
Proxima automatically resolves VM IP addresses using:

1. **QEMU Agent**: Primary method for real IP discovery
2. **Configurable Fallback**: `{vm_ip_base}.{vmid+100}` if QEMU Agent unavailable

### IP Base Configuration
```yaml
proxmox:
  vm_ip_base: "10.13.250"  # Base for VM IP generation
```

## Graceful vs Immediate Shutdown

Proxima provides two different shutdown methods:

### Graceful Shutdown
```bash
proxima 10.13.250.11 shutdown 100
```

**Features:**
- Sends ACPI shutdown signal to VM
- Waits for VM to gracefully shutdown
- Handles VM startup state (waits if VM is starting)
- Timeout: 60 seconds for shutdown completion
- Timeout: 120 seconds for VM startup completion
- Provides real-time feedback

**Process:**
1. Checks VM status
2. If VM is starting, waits up to 2 minutes for it to fully start
3. Sends graceful shutdown command
4. Monitors status for up to 60 seconds
5. Reports success or timeout

**Use Cases:**
- Production environments where data integrity is critical
- VMs running databases or applications that need clean shutdown
- Scheduled maintenance windows

### Immediate Stop
```bash
proxima 10.13.250.11 stop 100
```

**Features:**
- Immediately stops the VM (equivalent to power off)
- No waiting period
- Force termination
- Fast execution

**Use Cases:**
- Development environments
- Unresponsive VMs
- Emergency situations
- Quick restart cycles

### Error Handling

**Shutdown Command Errors:**
- `VM quit/powerdown failed - got timeout`: VM didn't respond to shutdown signal
- `timeout waiting for VM to fully start`: VM took too long to start before shutdown
- `timeout waiting for VM to shutdown`: VM didn't shutdown within 60 seconds

**Solutions:**
- Use `stop` command for immediate termination
- Check VM logs for shutdown issues
- Ensure QEMU Agent is installed and running
- Verify VM OS is responding to ACPI signals

## Advanced Features

### Idempotent Infrastructure
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

### SSH Key Management
Proxima supports multiple SSH authentication methods:

1. **Explicit Key Path**: Specify `key_path` in configuration
2. **Default Key Fallback**: If `key_path` not specified, Proxima searches for:
   - `~/.ssh/id_ed25519`
   - `~/.ssh/id_rsa`
   - `~/.ssh/id_ecdsa`
   - `~/.ssh/id_dsa`
3. **Password Authentication**: Use `password` field
4. **Key Copying**: Set `copy_local_key: true` to copy local key

### SSH Key Copy Feature
When `copy_local_key: true` is enabled:
- Proxima copies `~/.ssh/id_rsa.pub` from local machine
- Creates `.ssh` directory if it doesn't exist
- Sets proper permissions (600 for authorized_keys)
- Uses nested SSH (Proxmox host → target VM)

### VM-Specific SSH Configuration
Each VM can have its own SSH configuration that overrides global settings:

```yaml
vms:
  - name: "web-server"
    ssh:
      user: "ubuntu"           # VM-specific user
      password: "vm_password"  # VM-specific password
      key_path: "/path/to/vm/key"  # VM-specific key
      authorized_keys:          # Pre-authorized keys
        - "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."
```

## Troubleshooting

### Common Issues

#### 1. SSH Connection Failed
- Ensure SSH service is running on the VM
- Check if QEMU Agent is installed and running
- Verify network connectivity and firewall settings

#### 2. VM Creation Failed
- Verify Proxmox credentials and TLS configuration
- Check if VM ID is already in use
- Validate VLAN and network configuration

#### 3. IP Resolution Issues
- Ensure QEMU Agent is installed in the VM
- Check VM network configuration
- Verify bridge and VLAN settings

#### 4. Script Execution Failed
- Check script permissions and paths
- Ensure VM is accessible via SSH
- Verify script timeout settings

### Debug Mode

For detailed debugging, Proxima provides:
- Structured error messages with context
- Solution suggestions
- Debug information
- Stack traces when available

### Error Categories

- **Configuration Errors**: Invalid YAML, missing required fields
- **Connection Errors**: SSH/Proxmox API connectivity issues
- **VM Operation Errors**: Creation, start, stop failures
- **Authentication Errors**: SSH key or password issues

### Getting Help

```bash
# Show usage overview
proxima

# Show specific command help
proxima help list
proxima help start
proxima help shutdown
proxima help stop
proxima help delete
proxima help create
```

**Note:** Proxima does not use `-h` or `--help` flags. Use the `help` command instead.

## Best Practices

1. **Security**: Use SSH keys instead of passwords when possible
2. **Organization**: Use tags to categorize VMs (e.g., "web", "database", "development")
3. **Backup**: Keep your configuration file in version control
4. **Testing**: Test VM configurations in development environment first
5. **Idempotency**: Use config file mode for reproducible infrastructure
6. **Shutdown Strategy**: Use `shutdown` for production VMs, `stop` for development
7. **Command Structure**: Remember the new syntax: `proxima <host> <command> <vmid>`

## Architecture

Proxima follows clean architecture principles:

- **Domain Layer**: Core entities and business logic
- **Adapter Layer**: External integrations (Proxmox API, SSH, Config)
- **Service Layer**: Orchestrates operations between components
- **CLI Layer**: Command-line interface and user interaction

This design ensures maintainability, testability, and extensibility.