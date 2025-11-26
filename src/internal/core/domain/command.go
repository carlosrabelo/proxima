package domain

import (
	"time"
)

type Command struct {
	ID        int
	VMID      int
	Command   string
	Args      []string
	Status    CommandStatus
	Output    string
	Error     string
	StartTime time.Time
	EndTime   *time.Time
	Timeout   int
}

type CommandStatus string

const (
	CommandStatusPending   CommandStatus = "pending"
	CommandStatusRunning   CommandStatus = "running"
	CommandStatusCompleted CommandStatus = "completed"
	CommandStatusFailed    CommandStatus = "failed"
	CommandStatusTimeout   CommandStatus = "timeout"
)

func NewCommand(vmid int, command string, args []string, timeout int) *Command {
	return &Command{
		VMID:      vmid,
		Command:   command,
		Args:      args,
		Status:    CommandStatusPending,
		StartTime: time.Now(),
		Timeout:   timeout,
	}
}

func (c *Command) Start() {
	c.Status = CommandStatusRunning
}

func (c *Command) Complete(output string) {
	c.Status = CommandStatusCompleted
	c.Output = output
	now := time.Now()
	c.EndTime = &now
}

func (c *Command) Fail(err string) {
	c.Status = CommandStatusFailed
	c.Error = err
	now := time.Now()
	c.EndTime = &now
}

func (c *Command) TimeoutExceeded() {
	c.Status = CommandStatusTimeout
	c.Error = "Command execution timeout"
	now := time.Now()
	c.EndTime = &now
}

func (c *Command) Duration() time.Duration {
	if c.EndTime == nil {
		return time.Since(c.StartTime)
	}
	return c.EndTime.Sub(c.StartTime)
}
