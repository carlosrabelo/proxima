package proxmox

import (
	"context"
	"fmt"
	"strings"
)

// StartVM starts the guest via qm start (idempotent if already running).
func (c *Client) StartVM(ctx context.Context, vmid int) error {
	sess, err := c.SSH(ctx)
	if err != nil {
		return fmt.Errorf("vm start failed: %w", err)
	}
	defer sess.Close()

	statusOut, err := sess.CombinedOutput(ctx, fmt.Sprintf("qm status %d", vmid))
	if err == nil && strings.Contains(string(statusOut), "status: running") {
		return nil
	}

	if _, err := sess.CombinedOutput(ctx, fmt.Sprintf("qm start %d", vmid)); err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "already running") {
			return nil
		}
		return fmt.Errorf("vm start failed: %w", err)
	}
	return nil
}
