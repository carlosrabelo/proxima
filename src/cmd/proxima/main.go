package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"proxima/internal/adapters/config"
	"proxima/internal/adapters/proxmox"
	"proxima/internal/adapters/proxmox_ssh"
	"proxima/internal/adapters/ssh"
	"proxima/internal/core/ports"
	"proxima/internal/core/service"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	loginFlag bool

	// Command flags
	vmNameFlag  string
	commandFlag string
)

// Helper function to parse int
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// Helper function to show command help
func showCommandHelp(command string) {
	switch command {
	case "list":
		fmt.Println("Command: list")
		fmt.Println("Usage: proxima <host> list")
		fmt.Println("Description: List all VMs on the specified host")
	case "start":
		fmt.Println("Command: start")
		fmt.Println("Usage: proxima <host> start <vmid>")
		fmt.Println("Description: Start a VM with the specified ID")
	case "stop":
		fmt.Println("Command: stop")
		fmt.Println("Usage: proxima <host> stop <vmid>")
		fmt.Println("Description: Stop a VM with the specified ID")
	case "delete":
		fmt.Println("Command: delete")
		fmt.Println("Usage: proxima <host> delete <vmid>")
		fmt.Println("Description: Delete a VM with the specified ID")
	case "shutdown":
		fmt.Println("Command: shutdown")
		fmt.Println("Usage: proxima <host> shutdown <vmid>")
		fmt.Println("Description: Shutdown a VM gracefully with the specified ID")
	case "create":
		fmt.Println("Command: create")
		fmt.Println("Usage: proxima <host|yaml> create --name <name>")
		fmt.Println("Description: Create a VM from config")
		fmt.Println("Flags:")
		fmt.Println("  --name string    VM name (required)")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: list, start, stop, shutdown, delete, create")
	}
}

// Helper function to handle config file commands
func handleConfigCommand(configFile, command string, args []string) {
	switch command {
	case "create":
		if vmNameFlag == "" {
			fmt.Println("Error: --name flag is required")
			return
		}
		vmService, err := initializeServices(configFile, "")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if err := createVM(vmService, configFile, vmNameFlag); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	default:
		fmt.Printf("Error: unknown command '%s' for config file\n", command)
	}
}

// Helper function to handle host commands
func handleHostCommand(host, command string, args []string) {
	vmService, err := getVMService(host)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	switch command {
	case "list":
		if err := listVMs(vmService); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "start":
		if len(args) < 1 {
			fmt.Println("Error: vmid is required")
			fmt.Println("Usage: proxima <host> start <vmid>")
			return
		}
		vmid := parseInt(args[0])
		if vmid == 0 {
			fmt.Println("Error: invalid vmid")
			return
		}
		if err := startVM(vmService, vmid); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "stop":
		if len(args) < 1 {
			fmt.Println("Error: vmid is required")
			fmt.Println("Usage: proxima <host> stop <vmid>")
			return
		}
		vmid := parseInt(args[0])
		if vmid == 0 {
			fmt.Println("Error: invalid vmid")
			return
		}
		if err := stopVM(vmService, vmid); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "delete":
		if len(args) < 1 {
			fmt.Println("Error: vmid is required")
			fmt.Println("Usage: proxima <host> delete <vmid>")
			return
		}
		vmid := parseInt(args[0])
		if vmid == 0 {
			fmt.Println("Error: invalid vmid")
			return
		}
		if err := deleteVM(vmService, vmid); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "shutdown":
		if len(args) < 1 {
			fmt.Println("Error: vmid is required")
			fmt.Println("Usage: proxima <host> shutdown <vmid>")
			return
		}
		vmid := parseInt(args[0])
		if vmid == 0 {
			fmt.Println("Error: invalid vmid")
			return
		}
		if err := shutdownVM(vmService, vmid); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	case "create":
		if vmNameFlag == "" {
			fmt.Println("Error: --name flag is required")
			return
		}
		if err := createVM(vmService, "config.yaml", vmNameFlag); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		fmt.Println("Available commands: list, start, stop, shutdown, delete, create")
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "proxima",
		Short: "Tool for Proxmox VM management",
		Long: `Proxima is a CLI tool to create, manage and execute commands on Proxmox VMs via SSH.

USAGE:
  proxima <host> <command>
  proxima <yaml> <command>
  proxima help <command>

EXAMPLES:
  proxima 192.168.1.100 list
  proxima config.yaml create --name vm1`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Proxima - Tool for Proxmox VM management")
				fmt.Println()
				fmt.Println("Usage:")
				fmt.Println("  proxima <host> <command>")
				fmt.Println("  proxima <yaml> <command>")
				fmt.Println("  proxima help <command>")
				fmt.Println()
				fmt.Println("Commands:")
				fmt.Println("  list           List all VMs")
				fmt.Println("  start          Start a VM (requires <vmid>)")
				fmt.Println("  stop           Stop a VM immediately (requires <vmid>)")
				fmt.Println("  shutdown       Shutdown a VM gracefully (requires <vmid>)")
				fmt.Println("  delete         Delete a VM (requires <vmid>)")
				fmt.Println("  create         Create a VM from config (requires --name)")
				fmt.Println()
				fmt.Println("Global flags:")
				fmt.Println("  --login         Interactive login for host authentication")
				return
			}

			// Handle help command
			if args[0] == "help" {
				if len(args) < 2 {
					fmt.Println("Usage: proxima help <command>")
					return
				}
				showCommandHelp(args[1])
				return
			}

			// Handle regular commands
			if len(args) < 2 {
				fmt.Println("Error: missing command")
				fmt.Println("Usage: proxima <host|yaml> <command>")
				return
			}

			target := args[0]
			command := args[1]

			// Check if target is a YAML file
			if strings.HasSuffix(target, ".yaml") || strings.HasSuffix(target, ".yml") {
				handleConfigCommand(target, command, args[2:])
			} else {
				handleHostCommand(target, command, args[2:])
			}
		},
	}

	// Disable help flags completely
	rootCmd.SetHelpCommand(nil)
	rootCmd.DisableAutoGenTag = true
	rootCmd.DisableFlagsInUseLine = true
	rootCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {})

	rootCmd.PersistentFlags().BoolVar(&loginFlag, "login", false, "Interactive login for host authentication")
	rootCmd.PersistentFlags().StringVar(&vmNameFlag, "name", "", "VM name")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err.Error())
		os.Exit(1)
	}
}

// Helper to get VM service based on flags
func getVMService(host string) (ports.VMService, error) {
	if loginFlag {
		return initializeServicesWithLogin(host)
	}
	return initializeServicesWithSSHKey(host)
}

// --- Original Logic Functions ---

func runConfigFileMode(cmd *cobra.Command, args []string) error {
	configFile := "config.yaml"
	if len(args) > 0 {
		configFile = args[0]
	}
	fmt.Printf("Using config file: %s\n", configFile)
	vmService, err := initializeServices(configFile, "")
	if err != nil {
		return err
	}
	return runConfigMode(vmService)
}

func initializeServices(configFile, hostOverride string) (ports.VMService, error) {
	configAdapter := config.NewConfigAdapter()
	if err := configAdapter.LoadConfig(configFile); err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	// Validate configuration
	if err := configAdapter.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	proxmoxConfig := configAdapter.GetProxmoxConfig()
	sshConfig := configAdapter.GetSSHConfig()

	// Override host if provided via command line
	targetHost := hostOverride
	if targetHost == "" {
		targetHost = proxmoxConfig.Host
	}

	// For config file mode, use original Proxmox API adapter
	proxmoxAdapter := proxmox.NewProxmoxAdapterWithVMIPBase(
		targetHost,
		proxmoxConfig.Port,
		proxmoxConfig.User,
		proxmoxConfig.Password,
		proxmoxConfig.Node,
		proxmoxConfig.VMIPBase,
	)

	sshAdapter := ssh.NewSSHAdapterWithProxmox(
		sshConfig.User,
		sshConfig.Password,
		sshConfig.KeyPath,
		sshConfig.Port,
		sshConfig.CopyLocalKey,
		proxmoxAdapter,
	)

	vmService := service.NewVMService(proxmoxAdapter, sshAdapter)
	return vmService, nil
}

func initializeServicesWithSSHKey(host string) (ports.VMService, error) {
	// Create SSH direct adapter for Proxmox
	proxmoxSSHAdapter := proxmox_ssh.NewProxmoxSSHAdapter(host, "root", "", 22)

	sshAdapter := ssh.NewSSHAdapterWithProxmox(
		"root",
		"",
		"", // use default SSH keys
		22,
		true,
		proxmoxSSHAdapter,
	)

	vmService := service.NewVMService(proxmoxSSHAdapter, sshAdapter)
	return vmService, nil
}

func initializeServicesWithLogin(host string) (ports.VMService, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("SSH user: ")
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)

	fmt.Print("SSH password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Create SSH direct adapter for Proxmox
	proxmoxSSHAdapter := proxmox_ssh.NewProxmoxSSHAdapter(host, user, password, 22)

	sshAdapter := ssh.NewSSHAdapterWithProxmox(
		user,
		password,
		"", // use default SSH keys
		22,
		true,
		proxmoxSSHAdapter,
	)

	vmService := service.NewVMService(proxmoxSSHAdapter, sshAdapter)
	return vmService, nil
}

func runConfigMode(vmService ports.VMService) error {
	fmt.Println("Loading configuration...")

	// Load config to get VM definitions
	configAdapter := config.NewConfigAdapter()
	if err := configAdapter.LoadConfig("config.yaml"); err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	fmt.Println("Getting VM configurations...")

	// Get all VMs from config
	configVMs, err := configAdapter.GetAllVMConfigs()
	if err != nil {
		return fmt.Errorf("error getting VM configs: %w", err)
	}

	if len(configVMs) == 0 {
		fmt.Println("No VMs defined in config.yaml")
		return nil
	}

	fmt.Printf("Processing %d VM(s) from config.yaml...\n\n", len(configVMs))

	fmt.Println("Checking existing VMs...")
	// Get existing VMs from Proxmox
	existingVMs, err := vmService.ListVMs()
	if err != nil {
		return fmt.Errorf("error listing existing VMs: %w", err)
	}

	fmt.Printf("Found %d existing VMs\n\n", len(existingVMs))

	// Create map of existing VMs for quick lookup
	existingVMMap := make(map[int]bool)
	for _, vm := range existingVMs {
		existingVMMap[vm.ID] = true
	}

	// Process each VM from config
	for _, vmConfig := range configVMs {
		if existingVMMap[vmConfig.ID] {
			fmt.Printf("[OK] VM '%s' (ID: %d) already exists\n", vmConfig.Name, vmConfig.ID)
		} else {
			fmt.Printf("[CREATE] Creating VM '%s' (ID: %d)...\n", vmConfig.Name, vmConfig.ID)
			if err := vmService.CreateVM(vmConfig); err != nil {
				fmt.Printf("[ERROR] Failed to create VM '%s': %v\n", vmConfig.Name, err)
				continue
			}
			fmt.Printf("[OK] VM '%s' (ID: %d) created successfully\n", vmConfig.Name, vmConfig.ID)
		}
	}

	fmt.Println("\n[INFO] Current VM status:")
	return listVMs(vmService)
}

func createVM(vmService ports.VMService, configFile, vmName string) error {
	configAdapter := config.NewConfigAdapter()
	if err := configAdapter.LoadConfig(configFile); err != nil {
		return err
	}

	vm, err := configAdapter.GetVMConfig(vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found in configuration: %w", vmName, err)
	}

	fmt.Printf("Creating VM '%s' (ID: %d)...\n", vm.Name, vm.ID)
	if err := vmService.CreateVM(vm); err != nil {
		return fmt.Errorf("error creating VM: %w", err)
	}

	fmt.Printf("VM '%s' created successfully!\n", vm.Name)
	return nil
}

func startVM(vmService ports.VMService, vmid int) error {
	fmt.Printf("Starting VM %d...\n", vmid)
	if err := vmService.StartVM(vmid); err != nil {
		return fmt.Errorf("error starting VM: %w", err)
	}

	fmt.Printf("VM %d started successfully!\n", vmid)
	return nil
}

func stopVM(vmService ports.VMService, vmid int) error {
	fmt.Printf("Stopping VM %d...\n", vmid)
	if err := vmService.StopVM(vmid); err != nil {
		return fmt.Errorf("error stopping VM: %w", err)
	}

	fmt.Printf("VM %d stopped successfully!\n", vmid)
	return nil
}

func deleteVM(vmService ports.VMService, vmid int) error {
	fmt.Printf("Deleting VM %d...\n", vmid)
	if err := vmService.DeleteVM(vmid); err != nil {
		return fmt.Errorf("error deleting VM: %w", err)
	}

	fmt.Printf("VM %d deleted successfully!\n", vmid)
	return nil
}

func shutdownVM(vmService ports.VMService, vmid int) error {
	fmt.Printf("Shutting down VM %d...\n", vmid)
	if err := vmService.ShutdownVM(vmid); err != nil {
		return fmt.Errorf("error shutting down VM: %w", err)
	}

	fmt.Printf("VM %d shutdown successfully!\n", vmid)
	return nil
}

func listVMs(vmService ports.VMService) error {
	vms, err := vmService.ListVMs()
	if err != nil {
		return fmt.Errorf("error listing VMs: %w", err)
	}

	if len(vms) == 0 {
		fmt.Println("No VMs found.")
		return nil
	}

	fmt.Println("Found VMs:")
	fmt.Printf("%-6s %-20s %-10s %-8s %-8s\n", "VMID", "Name", "Status", "CPU", "Memory")
	fmt.Println("----------------------------------------------------")

	for _, vm := range vms {
		fmt.Printf("%-6d %-20s %-10s %-8d %-8d MB\n",
			vm.ID, vm.Name, vm.Status, vm.Cores, vm.Memory)
	}

	return nil
}
