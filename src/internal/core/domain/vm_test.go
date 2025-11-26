package domain

import (
	"testing"
	"time"
)

func TestNewVM(t *testing.T) {
	name := "test-vm"
	vmid := 100

	vm := NewVM(name, vmid)

	if vm.Name != name {
		t.Errorf("Expected name '%s', got '%s'", name, vm.Name)
	}

	if vm.ID != vmid {
		t.Errorf("Expected ID %d, got %d", vmid, vm.ID)
	}

	if vm.Status != VMStatusStopped {
		t.Errorf("Expected status '%s', got '%s'", VMStatusStopped, vm.Status)
	}

	if vm.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be set")
	}

	if vm.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}
}

func TestVM_Start(t *testing.T) {
	vm := NewVM("test-vm", 100)
	originalUpdatedAt := vm.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	vm.Start()

	if vm.Status != VMStatusRunning {
		t.Errorf("Expected status '%s', got '%s'", VMStatusRunning, vm.Status)
	}

	if !vm.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated after Start")
	}
}

func TestVM_Stop(t *testing.T) {
	vm := NewVM("test-vm", 100)
	vm.Start() // First start it
	originalUpdatedAt := vm.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	vm.Stop()

	if vm.Status != VMStatusStopped {
		t.Errorf("Expected status '%s', got '%s'", VMStatusStopped, vm.Status)
	}

	if !vm.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated after Stop")
	}
}

func TestVM_Delete(t *testing.T) {
	vm := NewVM("test-vm", 100)
	originalUpdatedAt := vm.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	vm.Delete()

	if vm.Status != VMStatusDeleting {
		t.Errorf("Expected status '%s', got '%s'", VMStatusDeleting, vm.Status)
	}

	if !vm.UpdatedAt.After(originalUpdatedAt) {
		t.Error("Expected UpdatedAt to be updated after Delete")
	}
}

func TestNewCommand(t *testing.T) {
	vmid := 100
	command := "ls"
	args := []string{"-la"}
	timeout := 30

	cmd := NewCommand(vmid, command, args, timeout)

	if cmd.VMID != vmid {
		t.Errorf("Expected VMID %d, got %d", vmid, cmd.VMID)
	}

	if cmd.Command != command {
		t.Errorf("Expected command '%s', got '%s'", command, cmd.Command)
	}

	if len(cmd.Args) != len(args) {
		t.Errorf("Expected %d args, got %d", len(args), len(cmd.Args))
	}

	for i, arg := range args {
		if cmd.Args[i] != arg {
			t.Errorf("Expected arg[%d] '%s', got '%s'", i, arg, cmd.Args[i])
		}
	}

	if cmd.Status != CommandStatusPending {
		t.Errorf("Expected status '%s', got '%s'", CommandStatusPending, cmd.Status)
	}

	if cmd.Timeout != timeout {
		t.Errorf("Expected timeout %d, got %d", timeout, cmd.Timeout)
	}

	if cmd.StartTime.IsZero() {
		t.Error("Expected StartTime to be set")
	}
}

func TestCommand_Start(t *testing.T) {
	cmd := NewCommand(100, "ls", []string{}, 30)

	cmd.Start()

	if cmd.Status != CommandStatusRunning {
		t.Errorf("Expected status '%s', got '%s'", CommandStatusRunning, cmd.Status)
	}
}

func TestCommand_Complete(t *testing.T) {
	cmd := NewCommand(100, "ls", []string{}, 30)
	cmd.Start()
	output := "file1\nfile2\n"

	cmd.Complete(output)

	if cmd.Status != CommandStatusCompleted {
		t.Errorf("Expected status '%s', got '%s'", CommandStatusCompleted, cmd.Status)
	}

	if cmd.Output != output {
		t.Errorf("Expected output '%s', got '%s'", output, cmd.Output)
	}

	if cmd.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestCommand_Fail(t *testing.T) {
	cmd := NewCommand(100, "ls", []string{}, 30)
	cmd.Start()
	errorMsg := "file not found"

	cmd.Fail(errorMsg)

	if cmd.Status != CommandStatusFailed {
		t.Errorf("Expected status '%s', got '%s'", CommandStatusFailed, cmd.Status)
	}

	if cmd.Error != errorMsg {
		t.Errorf("Expected error '%s', got '%s'", errorMsg, cmd.Error)
	}

	if cmd.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestCommand_TimeoutExceeded(t *testing.T) {
	cmd := NewCommand(100, "sleep", []string{"10"}, 1)
	cmd.Start()

	cmd.TimeoutExceeded()

	if cmd.Status != CommandStatusTimeout {
		t.Errorf("Expected status '%s', got '%s'", CommandStatusTimeout, cmd.Status)
	}

	if cmd.Error != "Command execution timeout" {
		t.Errorf("Expected error 'Command execution timeout', got '%s'", cmd.Error)
	}

	if cmd.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestCommand_Duration(t *testing.T) {
	cmd := NewCommand(100, "ls", []string{}, 30)

	// Test duration when command is still running
	duration := cmd.Duration()
	if duration <= 0 {
		t.Error("Expected positive duration for running command")
	}

	// Test duration after completion
	cmd.Start()
	time.Sleep(1 * time.Millisecond)
	cmd.Complete("output")
	duration = cmd.Duration()
	if duration <= 0 {
		t.Error("Expected positive duration for completed command")
	}
}
