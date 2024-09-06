package upload_test

import (
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
