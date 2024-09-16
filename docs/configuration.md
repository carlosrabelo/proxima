# Configuration

Proxima loads a YAML (or JSON) manifesto for each VM.

## Default path

When `-c` is omitted, Proxima looks for:

1. `./Proxima.yaml`
2. `./vms/Proxima.yaml`

The `vms/` directory is local-only (gitignored except `.gitkeep`). Copy the example into place:

```bash
cp docs/proxima.example.yaml vms/Proxima.yaml
```

Then edit secrets and host settings in `vms/Proxima.yaml`. Prefer env vars for secrets when possible: `PROXIMA_PASSWORD`, `PROXIMA_TOKEN_SECRET`, `PROXIMA_ROOT_PASSWORD`.

## Example file

See [`proxima.example.yaml`](proxima.example.yaml) in this directory.

## Auth modes

Use exactly one:

- Password: `user` + `password`
- API token: `token_id` + `token_secret` (also set `ssh_private_key_path` for SSH)

## Guest SSH public key

Optional `guest_ssh_public_key_path` (or `--guest-ssh-public-key`) points at an OpenSSH `.pub` file. During `proxima build` / `create`, that key is written to `/root/.ssh/authorized_keys` inside the Alpine image. A leading `~/` is expanded. Rebuild the qcow2 and recreate the VM after changing this path.

## Useful flags

- `--start` / `--wait-ip` (defaults true on `create` / `start`)
- `--agent-timeout` (default `120s`)
- `--agent-poll-interval` (default `2s`)
- `--guest-ssh-public-key` path to bake a pubkey into the guest image
- `-c path` to force a non-default manifesto
