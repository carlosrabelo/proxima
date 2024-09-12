package cli

import (
	"fmt"
	"os"
)

// resolveConfigPath returns the config file to load.
// If explicit is non-empty it wins. Otherwise Proxima.yaml in the CWD is
// tried, then vms/Proxima.yaml (project convention).
func resolveConfigPath(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	for _, candidate := range []string{"Proxima.yaml", "vms/Proxima.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("config manifesto not found: create Proxima.yaml or vms/Proxima.yaml, or pass -c")
}
