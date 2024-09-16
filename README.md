# Proxima

Proxima is a Go CLI that builds an Alpine Linux Docker-ready qcow2 image locally with Docker, then uploads and provisions it as a VM on Proxmox VE.

## Highlights

- Local Alpine qcow2 build via Docker and `alpine-make-vm-image`
- Proxmox provision through SSH `qm` (create, importdisk, attach, boot, agent)
- Start the VM and resolve guest IPv4 via the QEMU Guest Agent (`WaitIP`)
- Config file plus CLI flags for hosts, auth, VM sizing, and storage
- Stage status lines and cleanup of partial artifacts on failure

## Prerequisites

- **Go 1.23+** — required to build from source
- **Docker** — host CLI for the local qcow2 build
- **SSH** reachability to the Proxmox node
- **Proxmox VE** with `qm`, storage id (default `local-lvm`), and bridge (default `vmbr0`)
- DHCP (or other IP assignment) on the guest bridge so `WaitIP` can observe an address

## Installation

### Build from Source

```bash
git clone <repo-url>
cd proxima
make build
make install         # ~/.local/bin/proxima
make uninstall
make package         # .deb into dist/ (binary at /usr/bin)
sudo dpkg -i dist/*.deb
```

Binary after build: `bin/proxima`.

## Usage

Build a local qcow2 only:

```bash
proxima build
```

Full pipeline (Validate → BuildLocal → Upload → Provision → Start → WaitIP by default):

```bash
proxima create
```

Start an already-provisioned VM and print guest IPv4:

```bash
proxima start
```

Wait for / print guest IPv4 without starting:

```bash
proxima ip
```

## Configuration

Copy [`docs/proxima.example.yaml`](docs/proxima.example.yaml) to `vms/Proxima.yaml` (local only; `vms/` is gitignored except `.gitkeep`). That manifesto is read automatically when `-c` is omitted. Details: [`docs/configuration.md`](docs/configuration.md).

Discovery order without `-c`: `./Proxima.yaml`, then `./vms/Proxima.yaml`. Pass `-c path` only to override. Put the built disk under `vms/` (for example `output: "vms/alpine-ready.qcow2"`).

Authentication (exactly one mode):

- Password: `user` / `password` (or env `PROXIMA_PASSWORD`)
- API token: `token_id` / `token_secret` (or env `PROXIMA_TOKEN_SECRET`); token mode also needs `ssh_private_key_path`

Guest root password: `root_password` or env `PROXIMA_ROOT_PASSWORD`.

Optional guest SSH: `guest_ssh_public_key_path` (or `--guest-ssh-public-key`) embeds an OpenSSH `.pub` into `/root/.ssh/authorized_keys` at image build time. Rebuild the qcow2 and recreate the VM after changing it.

Create / start flags: `--start` / `--wait-ip` (defaults true), `--agent-timeout` (default `120s`), `--agent-poll-interval` (default `2s`).

Validate (from the repo root):

```bash
proxima config check
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
make build           # Compile binary to bin/proxima
make test            # Run all tests
make quality         # Format check, vet, test, and build
make install         # Install binary to ~/.local/bin
make uninstall       # Remove from ~/.local/bin
make package         # Build .deb into dist/ (binary at /usr/bin)
make deb             # Alias for package
```

## License

This project is licensed under the GPL-3.0 License — see [LICENSE](LICENSE) for details.
