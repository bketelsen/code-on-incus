package health

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mensfeld/code-on-incus/internal/config"
	"github.com/mensfeld/code-on-incus/internal/container"
	"github.com/mensfeld/code-on-incus/internal/network"
	"github.com/mensfeld/code-on-incus/internal/session"
	"github.com/mensfeld/code-on-incus/internal/tool"
)

// CheckIncus verifies that Incus is available and running
func CheckIncus() HealthCheck {
	// Check if incus binary exists
	if _, err := exec.LookPath("incus"); err != nil {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusFailed,
			Message: "Incus binary not found",
		}
	}

	// Check if Incus is available (daemon running and accessible)
	if !container.Available() {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusFailed,
			Message: "Incus daemon not running or not accessible",
		}
	}

	// Get Incus version
	versionOutput, err := container.IncusOutput("version")
	if err != nil {
		return HealthCheck{
			Name:    "incus",
			Status:  StatusOK,
			Message: "Running (version unknown)",
		}
	}

	// Parse version - extract server version from multi-line output
	// Example output: "Client version: 6.20\nServer version: 6.20"
	version := strings.TrimSpace(versionOutput)
	for _, line := range strings.Split(version, "\n") {
		if strings.HasPrefix(line, "Server version:") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Server version:"))
			break
		}
	}

	return HealthCheck{
		Name:    "incus",
		Status:  StatusOK,
		Message: fmt.Sprintf("Running (version %s)", version),
		Details: map[string]interface{}{
			"version": version,
		},
	}
}

// CheckPermissions verifies user has correct group membership
func CheckPermissions() HealthCheck {
	// On macOS, no group check needed
	if runtime.GOOS == "darwin" {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusOK,
			Message: "macOS - no group required",
		}
	}

	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine current user: %v", err),
		}
	}

	// Get user's groups
	groups, err := currentUser.GroupIds()
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not determine user groups: %v", err),
		}
	}

	// Look for incus-admin group
	incusGroup, err := user.LookupGroup("incus-admin")
	if err != nil {
		return HealthCheck{
			Name:    "permissions",
			Status:  StatusFailed,
			Message: "incus-admin group not found",
		}
	}

	// Check if user is in the group
	for _, gid := range groups {
		if gid == incusGroup.Gid {
			return HealthCheck{
				Name:    "permissions",
				Status:  StatusOK,
				Message: "User in incus-admin group",
				Details: map[string]interface{}{
					"user":  currentUser.Username,
					"group": "incus-admin",
				},
			}
		}
	}

	return HealthCheck{
		Name:    "permissions",
		Status:  StatusFailed,
		Message: fmt.Sprintf("User '%s' not in incus-admin group", currentUser.Username),
	}
}

// CheckImage verifies that the default image exists
func CheckImage(imageName string) HealthCheck {
	if imageName == "" {
		imageName = "coi"
	}

	exists, err := container.ImageExists(imageName)
	if err != nil {
		return HealthCheck{
			Name:    "image",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check image: %v", err),
		}
	}

	if !exists {
		return HealthCheck{
			Name:    "image",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Image '%s' not found (run 'coi build')", imageName),
			Details: map[string]interface{}{
				"expected": imageName,
			},
		}
	}

	// Get image fingerprint
	output, err := container.IncusOutput("image", "list", imageName, "--format=csv", "-c", "f")
	fingerprint := ""
	if err == nil && output != "" {
		lines := strings.Split(output, "\n")
		if len(lines) > 0 {
			fingerprint = strings.TrimSpace(lines[0])
			if len(fingerprint) > 12 {
				fingerprint = fingerprint[:12] + "..."
			}
		}
	}

	message := imageName
	if fingerprint != "" {
		message = fmt.Sprintf("%s (fingerprint: %s)", imageName, fingerprint)
	}

	return HealthCheck{
		Name:    "image",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"alias":       imageName,
			"fingerprint": fingerprint,
		},
	}
}

// CheckNetworkBridge verifies the network bridge is configured
func CheckNetworkBridge() HealthCheck {
	// Get default profile to find network device
	output, err := container.IncusOutput("profile", "device", "show", "default")
	if err != nil {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not get default profile: %v", err),
		}
	}

	// Parse network name from profile (looking for eth0 device)
	var networkName string
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "eth0:" {
			// Look for network: line
			for j := i + 1; j < len(lines) && j < i+10; j++ {
				if strings.Contains(lines[j], "network:") {
					parts := strings.Split(lines[j], ":")
					if len(parts) >= 2 {
						networkName = strings.TrimSpace(parts[1])
						break
					}
				}
			}
			break
		}
	}

	if networkName == "" {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusFailed,
			Message: "No eth0 network device in default profile",
		}
	}

	// Get network configuration
	networkOutput, err := container.IncusOutput("network", "show", networkName)
	if err != nil {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not get network info for %s: %v", networkName, err),
		}
	}

	// Parse IPv4 address
	var ipv4Address string
	for _, line := range strings.Split(networkOutput, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ipv4.address:") {
			ipv4Address = strings.TrimSpace(strings.TrimPrefix(line, "ipv4.address:"))
			break
		}
	}

	if ipv4Address == "" || ipv4Address == "none" {
		return HealthCheck{
			Name:    "network_bridge",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s has no IPv4 address", networkName),
		}
	}

	return HealthCheck{
		Name:    "network_bridge",
		Status:  StatusOK,
		Message: fmt.Sprintf("%s (%s)", networkName, ipv4Address),
		Details: map[string]interface{}{
			"name": networkName,
			"ipv4": ipv4Address,
		},
	}
}

// CheckIPForwarding verifies IP forwarding is enabled
func CheckIPForwarding() HealthCheck {
	// On macOS, IP forwarding works differently
	if runtime.GOOS == "darwin" {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusOK,
			Message: "macOS - managed by Incus",
		}
	}

	// Read /proc/sys/net/ipv4/ip_forward
	content, err := os.ReadFile("/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Could not check: %v", err),
		}
	}

	value := strings.TrimSpace(string(content))
	if value == "1" {
		return HealthCheck{
			Name:    "ip_forwarding",
			Status:  StatusOK,
			Message: "Enabled",
		}
	}

	return HealthCheck{
		Name:    "ip_forwarding",
		Status:  StatusWarning,
		Message: "Disabled (may affect container networking)",
	}
}

// CheckFirewall verifies firewalld availability based on network mode
func CheckFirewall(mode config.NetworkMode) HealthCheck {
	available := network.FirewallAvailable()

	if mode == config.NetworkModeOpen {
		// Firewall not required for open mode
		if available {
			return HealthCheck{
				Name:    "firewall",
				Status:  StatusOK,
				Message: "Available (not required for open mode)",
			}
		}
		return HealthCheck{
			Name:    "firewall",
			Status:  StatusOK,
			Message: "Not available (not required for open mode)",
		}
	}

	// Required for restricted/allowlist modes
	if !available {
		return HealthCheck{
			Name:    "firewall",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Not available (required for %s mode)", mode),
		}
	}

	return HealthCheck{
		Name:    "firewall",
		Status:  StatusOK,
		Message: fmt.Sprintf("Running (%s mode available)", mode),
	}
}

// CheckCOIDirectory verifies the COI directory exists and is writable
func CheckCOIDirectory() HealthCheck {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "coi_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	coiDir := filepath.Join(homeDir, ".coi")

	// Check if directory exists
	info, err := os.Stat(coiDir)
	if os.IsNotExist(err) {
		return HealthCheck{
			Name:    "coi_directory",
			Status:  StatusWarning,
			Message: fmt.Sprintf("%s does not exist (will be created on first run)", coiDir),
		}
	}
	if err != nil {
		return HealthCheck{
			Name:    "coi_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not access %s: %v", coiDir, err),
		}
	}

	if !info.IsDir() {
		return HealthCheck{
			Name:    "coi_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not a directory", coiDir),
		}
	}

	// Check if writable by creating a temp file
	testFile := filepath.Join(coiDir, ".health-check-test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return HealthCheck{
			Name:    "coi_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not writable", coiDir),
		}
	}
	os.Remove(testFile)

	return HealthCheck{
		Name:    "coi_directory",
		Status:  StatusOK,
		Message: "~/.coi (writable)",
		Details: map[string]interface{}{
			"path": coiDir,
		},
	}
}

// CheckSessionsDirectory verifies the sessions directory exists and is writable
func CheckSessionsDirectory(cfg *config.Config) HealthCheck {
	// Get configured tool to determine sessions directory
	toolName := cfg.Tool.Name
	if toolName == "" {
		toolName = "claude"
	}
	toolInstance, err := tool.Get(toolName)
	if err != nil {
		toolInstance = tool.GetDefault()
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not determine home directory: %v", err),
		}
	}

	baseDir := filepath.Join(homeDir, ".coi")
	sessionsDir := session.GetSessionsDir(baseDir, toolInstance)

	// Check if directory exists
	info, err := os.Stat(sessionsDir)
	if os.IsNotExist(err) {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusOK,
			Message: fmt.Sprintf("%s (will be created)", filepath.Base(sessionsDir)),
			Details: map[string]interface{}{
				"path": sessionsDir,
			},
		}
	}
	if err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("Could not access %s: %v", sessionsDir, err),
		}
	}

	if !info.IsDir() {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not a directory", sessionsDir),
		}
	}

	// Check if writable
	testFile := filepath.Join(sessionsDir, ".health-check-test")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		return HealthCheck{
			Name:    "sessions_directory",
			Status:  StatusFailed,
			Message: fmt.Sprintf("%s is not writable", sessionsDir),
		}
	}
	os.Remove(testFile)

	return HealthCheck{
		Name:    "sessions_directory",
		Status:  StatusOK,
		Message: fmt.Sprintf("~/.coi/%s (writable)", filepath.Base(sessionsDir)),
		Details: map[string]interface{}{
			"path": sessionsDir,
		},
	}
}

// CheckConfiguration verifies the configuration is loaded correctly
func CheckConfiguration(cfg *config.Config) HealthCheck {
	if cfg == nil {
		return HealthCheck{
			Name:    "config",
			Status:  StatusFailed,
			Message: "Configuration not loaded",
		}
	}

	// Find which config files exist
	configPaths := config.GetConfigPaths()
	var loadedFrom []string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			loadedFrom = append(loadedFrom, path)
		}
	}

	message := "Defaults only (no config files)"
	if len(loadedFrom) > 0 {
		message = loadedFrom[len(loadedFrom)-1] // Show highest priority
	}

	return HealthCheck{
		Name:    "config",
		Status:  StatusOK,
		Message: message,
		Details: map[string]interface{}{
			"loaded_from": loadedFrom,
		},
	}
}

// CheckNetworkMode reports the configured network mode
func CheckNetworkMode(mode config.NetworkMode) HealthCheck {
	if mode == "" {
		mode = config.NetworkModeRestricted
	}

	return HealthCheck{
		Name:    "network_mode",
		Status:  StatusOK,
		Message: string(mode),
		Details: map[string]interface{}{
			"mode": string(mode),
		},
	}
}

// CheckTool reports the configured tool
func CheckTool(toolName string) HealthCheck {
	if toolName == "" {
		toolName = "claude"
	}

	_, err := tool.Get(toolName)
	if err != nil {
		return HealthCheck{
			Name:    "tool",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Unknown tool: %s", toolName),
		}
	}

	return HealthCheck{
		Name:    "tool",
		Status:  StatusOK,
		Message: toolName,
		Details: map[string]interface{}{
			"name": toolName,
		},
	}
}

// CheckDNS verifies DNS resolution is working
func CheckDNS() HealthCheck {
	// Try to resolve a well-known domain
	testDomain := "api.anthropic.com"

	ips, err := net.LookupIP(testDomain)
	if err != nil {
		return HealthCheck{
			Name:    "dns_resolution",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Failed to resolve %s: %v", testDomain, err),
		}
	}

	if len(ips) == 0 {
		return HealthCheck{
			Name:    "dns_resolution",
			Status:  StatusWarning,
			Message: fmt.Sprintf("No IPs found for %s", testDomain),
		}
	}

	return HealthCheck{
		Name:    "dns_resolution",
		Status:  StatusOK,
		Message: fmt.Sprintf("Working (%s -> %d IPs)", testDomain, len(ips)),
		Details: map[string]interface{}{
			"test_domain": testDomain,
			"ip_count":    len(ips),
		},
	}
}

// CheckPasswordlessSudo verifies passwordless sudo for firewall-cmd
func CheckPasswordlessSudo() HealthCheck {
	// On macOS, not needed
	if runtime.GOOS == "darwin" {
		return HealthCheck{
			Name:    "passwordless_sudo",
			Status:  StatusOK,
			Message: "macOS - not required",
		}
	}

	// Try to run firewall-cmd --state with sudo -n
	cmd := exec.Command("sudo", "-n", "firewall-cmd", "--state")
	err := cmd.Run()
	if err != nil {
		// Check if firewall-cmd even exists
		if _, lookErr := exec.LookPath("firewall-cmd"); lookErr != nil {
			return HealthCheck{
				Name:    "passwordless_sudo",
				Status:  StatusOK,
				Message: "firewall-cmd not installed (not needed for open mode)",
			}
		}

		return HealthCheck{
			Name:    "passwordless_sudo",
			Status:  StatusWarning,
			Message: "Passwordless sudo not configured for firewall-cmd",
		}
	}

	return HealthCheck{
		Name:    "passwordless_sudo",
		Status:  StatusOK,
		Message: "Configured for firewall-cmd",
	}
}
