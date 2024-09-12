# Proxima

Proxima is a Go CLI that builds an Alpine Linux Docker-ready qcow2 image locally with Docker, then uploads and provisions it as a VM on Proxmox VE.

## Highlights

- Local Alpine qcow2 build via Docker and `alpine-make-vm-image`
- Proxmox provision through SSH `qm` (create, importdisk, attach, boot, agent)
- Config file plus CLI flags for hosts, auth, VM sizing, and storage
- Stage status lines and cleanup of partial artifacts on failure

## Requirements

- Go 1.23 or newer
- Docker (host CLI)
- SSH reachability to the Proxmox node
- Proxmox VE with `qm`, storage id (default `local-lvm`), and bridge (default `vmbr0`)

## Installation

```bash
make build
```

Binary: `bin/proxima`.

## Configuration

Copy and edit [`cfg/proxima.example.yaml`](cfg/proxima.example.yaml).

Authentication (exactly one mode):

- Password: `user` / `password` (or env `PROXIMA_PASSWORD`)
- API token: `token_id` / `token_secret` (or env `PROXIMA_TOKEN_SECRET`); token mode also needs `ssh_private_key_path`

Guest root password: `root_password` or env `PROXIMA_ROOT_PASSWORD`.

Validate:

```bash
proxima config check -c cfg/proxima.example.yaml
```

## Usage

Build a local qcow2 only:

```bash
proxima build -c cfg/proxima.example.yaml
```

Full pipeline (Validate → BuildLocal → Upload → Provision → Start → WaitIP by default):

```bash
proxima create -c cfg/proxima.example.yaml
```

## Pipeline stages

| Stage | Role |
| --- | --- |
| Validate | Load and validate configuration |
| BuildLocal | Build Alpine Docker-ready qcow2 locally |
| Upload | SCP/SFTP disk to `/var/tmp/proxima/<vmid>-disk.qcow2` |
| Provision | `qm create` / `importdisk` / attach scsi0 / boot / agent |
| Start | Start the VM |
| WaitIP | Resolve guest IPv4 via QEMU Guest Agent |

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Usage / validation error |
| 2 | Local build / Docker failure |
| 3 | Proxmox transport or API failure |
| 4 | Partial-provision cleanup failure |

## Development

```bash
make test
make quality
```

## License

License: TBD
