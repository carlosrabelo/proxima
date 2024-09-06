package upload

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"proxima/proxima/internal/proxmox"
)

const remoteDir = "/var/tmp/proxima"

// Session is an SSH session that can run commands and upload files.
type Session interface {
	CombinedOutput(ctx context.Context, command string) ([]byte, error)
	Upload(ctx context.Context, localPath, remotePath string) error
	Close() error
}

// Uploader transfers a local qcow2 to the Proxmox node over SSH.
type Uploader struct {
	Client *proxmox.Client

	// OpenSSH is an optional test seam; when nil, Client.SSH is used.
	OpenSSH func(ctx context.Context) (Session, error)
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

// UploadQcow2 copies localPath to the Proxmox staging path for vmid.
func (u *Uploader) UploadQcow2(ctx context.Context, localPath string, vmid int) (Result, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Result{}, fmt.Errorf("disk upload failed: local file not found: %s", localPath)
		}
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}
	localSize := info.Size()
	remote := RemotePath(vmid)

	sess, err := u.open(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}
	defer sess.Close()

	if _, err := sess.CombinedOutput(ctx, "mkdir -p "+remoteDir); err != nil {
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}

	if err := sess.Upload(ctx, localPath, remote); err != nil {
		removeRemoteBestEffort(ctx, sess, remote)
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}

	out, err := sess.CombinedOutput(ctx, "stat -c%s "+remote)
	if err != nil {
		removeRemoteBestEffort(ctx, sess, remote)
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}
	remoteSize, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		removeRemoteBestEffort(ctx, sess, remote)
		return Result{}, fmt.Errorf("disk upload failed: %w", err)
	}
	if remoteSize != localSize {
		removeRemoteBestEffort(ctx, sess, remote)
		return Result{}, fmt.Errorf("disk upload failed: upload size mismatch: local %d remote %d", localSize, remoteSize)
	}

	return Result{
		LocalPath:  localPath,
		RemotePath: remote,
		Bytes:      localSize,
	}, nil
}

func removeRemoteBestEffort(ctx context.Context, sess Session, remotePath string) {
	_, _ = sess.CombinedOutput(ctx, "rm -f "+remotePath)
}

func (u *Uploader) open(ctx context.Context) (Session, error) {
	if u.OpenSSH != nil {
		return u.OpenSSH(ctx)
	}
	if u.Client == nil {
		return nil, fmt.Errorf("proxmox client is nil")
	}
	sess, err := u.Client.SSH(ctx)
	if err != nil {
		return nil, err
	}
	return &cmdOnlySession{sess: sess}, nil
}

// cmdOnlySession adapts proxmox.SSHSession when file upload is not available on the session.
type cmdOnlySession struct {
	sess proxmox.SSHSession
}

func (s *cmdOnlySession) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	return s.sess.CombinedOutput(ctx, command)
}

func (s *cmdOnlySession) Upload(ctx context.Context, localPath, remotePath string) error {
	_ = ctx
	_ = localPath
	_ = remotePath
	return fmt.Errorf("scp upload not implemented")
}

func (s *cmdOnlySession) Close() error {
	return s.sess.Close()
}
