package upload

import (
	"fmt"

	"proxima/proxima/internal/proxmox"
)

const remoteDir = "/var/tmp/proxima"

// Uploader transfers a local qcow2 to the Proxmox node over SSH.
type Uploader struct {
	Client *proxmox.Client
}

// Result is the outcome of a successful disk upload.
type Result struct {
	LocalPath  string
	RemotePath string
	Bytes      int64
}

// RemotePath returns the staged remote qcow2 path for vmid.
func RemotePath(vmid int) string {
	return fmt.Sprintf("%s/%d-disk.qcow2", remoteDir, vmid)
}
