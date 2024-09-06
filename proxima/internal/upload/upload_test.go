package upload_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"proxima/proxima/internal/upload"
)

func TestUploadRemotePathUsesVMID(t *testing.T) {
	got := upload.RemotePath(200)
	want := "/var/tmp/proxima/200-disk.qcow2"
	if got != want {
		t.Fatalf("RemotePath(200) = %q, want %q", got, want)
	}

	got = upload.RemotePath(999)
	want = "/var/tmp/proxima/999-disk.qcow2"
	if got != want {
		t.Fatalf("RemotePath(999) = %q, want %q", got, want)
	}
}

func TestUploadCreatesRemoteDir(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "disk.qcow2")
	payload := []byte("qcow2-payload")
	if err := os.WriteFile(local, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeSession{
		files: map[string][]byte{},
		sizes: map[string]int64{},
	}
	u := &upload.Uploader{
		OpenSSH: func(ctx context.Context) (upload.Session, error) {
			return fake, nil
		},
	}

	res, err := u.UploadQcow2(context.Background(), local, 200)
	if err != nil {
		t.Fatalf("UploadQcow2: %v", err)
	}

	if len(fake.commands) == 0 || fake.commands[0] != "mkdir -p /var/tmp/proxima" {
		t.Fatalf("commands = %#v, want mkdir first", fake.commands)
	}
	remote := "/var/tmp/proxima/200-disk.qcow2"
	if res.RemotePath != remote {
		t.Fatalf("RemotePath = %q, want %q", res.RemotePath, remote)
	}
	if res.Bytes != int64(len(payload)) {
		t.Fatalf("Bytes = %d, want %d", res.Bytes, len(payload))
	}
	if string(fake.files[remote]) != string(payload) {
		t.Fatalf("remote content = %q, want %q", fake.files[remote], payload)
	}
	foundStat := false
	for _, cmd := range fake.commands {
		if cmd == "stat -c%s "+remote {
			foundStat = true
			break
		}
	}
	if !foundStat {
		t.Fatalf("missing stat command in %#v", fake.commands)
	}
}

func TestUploadSizeMismatchCleansRemote(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "disk.qcow2")
	if err := os.WriteFile(local, []byte("local-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &fakeSession{
		files:     map[string][]byte{},
		sizes:     map[string]int64{},
		forceSize: 3,
	}
	u := &upload.Uploader{
		OpenSSH: func(ctx context.Context) (upload.Session, error) {
			return fake, nil
		},
	}

	remote := "/var/tmp/proxima/42-disk.qcow2"
	_, err := u.UploadQcow2(context.Background(), local, 42)
	if err == nil {
		t.Fatal("expected size mismatch error")
	}
	want := "disk upload failed: upload size mismatch: local 11 remote 3"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
	if _, ok := fake.files[remote]; ok {
		t.Fatalf("remote file %q should have been removed", remote)
	}
	foundRM := false
	for _, cmd := range fake.commands {
		if cmd == "rm -f "+remote {
			foundRM = true
			break
		}
	}
	if !foundRM {
		t.Fatalf("missing rm command in %#v", fake.commands)
	}
}

type fakeSession struct {
	commands  []string
	files     map[string][]byte
	sizes     map[string]int64
	forceSize int64 // when non-zero, stat returns this size
	closed    bool
}

func (f *fakeSession) CombinedOutput(ctx context.Context, command string) ([]byte, error) {
	_ = ctx
	f.commands = append(f.commands, command)
	switch {
	case strings.HasPrefix(command, "mkdir -p "):
		return nil, nil
	case strings.HasPrefix(command, "stat -c%s "):
		path := strings.TrimPrefix(command, "stat -c%s ")
		if f.forceSize != 0 {
			return []byte(fmt.Sprintf("%d\n", f.forceSize)), nil
		}
		size, ok := f.sizes[path]
		if !ok {
			return nil, fmt.Errorf("stat: no such file")
		}
		return []byte(fmt.Sprintf("%d\n", size)), nil
	case strings.HasPrefix(command, "rm -f "):
		path := strings.TrimPrefix(command, "rm -f ")
		delete(f.files, path)
		delete(f.sizes, path)
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected command: %s", command)
	}
}

func (f *fakeSession) Upload(ctx context.Context, localPath, remotePath string) error {
	_ = ctx
	data, err := os.ReadFile(localPath)
	if err != nil {
		return err
	}
	f.files[remotePath] = data
	f.sizes[remotePath] = int64(len(data))
	return nil
}

func (f *fakeSession) Close() error {
	f.closed = true
	return nil
}
