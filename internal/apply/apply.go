package apply

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/muuvmuuv/php-tuner/internal/calculator"
)

// Common PHP-FPM pool configuration paths
var configPaths = []string{
	// Debian/Ubuntu
	"/etc/php/8.3/fpm/pool.d/www.conf",
	"/etc/php/8.2/fpm/pool.d/www.conf",
	"/etc/php/8.1/fpm/pool.d/www.conf",
	"/etc/php/8.0/fpm/pool.d/www.conf",
	"/etc/php/7.4/fpm/pool.d/www.conf",
	// RHEL/CentOS/Fedora
	"/etc/php-fpm.d/www.conf",
	// Generic
	"/etc/php-fpm.conf",
	// macOS (Homebrew)
	"/opt/homebrew/etc/php/8.3/php-fpm.d/www.conf",
	"/opt/homebrew/etc/php/8.2/php-fpm.d/www.conf",
	"/usr/local/etc/php/8.3/php-fpm.d/www.conf",
	"/usr/local/etc/php/8.2/php-fpm.d/www.conf",
}

// Common PHP-FPM service names
var serviceNames = []string{
	"php-fpm",
	"php8.3-fpm",
	"php8.2-fpm",
	"php8.1-fpm",
	"php8.0-fpm",
	"php7.4-fpm",
}

// Result holds the result of an apply operation
type Result struct {
	ConfigPath  string
	BackupPath  string
	ServiceName string
	Restarted   bool
	Changes     []string
}

// FindConfigFile locates the PHP-FPM pool configuration
func FindConfigFile() (string, error) {
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("could not find PHP-FPM pool configuration file")
}

// FindServiceName finds the active PHP-FPM service
func FindServiceName() string {
	for _, name := range serviceNames {
		var cmd *exec.Cmd

		switch runtime.GOOS {
		case "linux":
			cmd = exec.Command("systemctl", "is-active", "--quiet", name)
		case "darwin":
			cmd = exec.Command("brew", "services", "list")
		default:
			continue
		}

		if runtime.GOOS == "darwin" {
			output, err := cmd.Output()
			if err == nil && strings.Contains(string(output), "php") && strings.Contains(string(output), "started") {
				// Extract the service name from brew services list
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					if strings.Contains(line, "php") && strings.Contains(line, "started") {
						fields := strings.Fields(line)
						if len(fields) > 0 {
							return fields[0]
						}
					}
				}
			}
		} else {
			if err := cmd.Run(); err == nil {
				return name
			}
		}
	}
	return ""
}

// Apply writes the configuration to the PHP-FPM config file
func Apply(cfg *calculator.Config, configPath string, restart bool) (*Result, error) {
	result := &Result{
		ConfigPath: configPath,
	}

	// Read existing config
	content, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Create backup
	result.BackupPath = configPath + ".backup"
	if err := os.WriteFile(result.BackupPath, content, 0644); err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	// Update configuration
	newContent, changes := updateConfig(string(content), cfg)
	result.Changes = changes

	// Write new config
	if err := os.WriteFile(configPath, []byte(newContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	// Restart service if requested
	if restart {
		result.ServiceName = FindServiceName()
		if result.ServiceName != "" {
			if err := restartService(result.ServiceName); err != nil {
				return result, fmt.Errorf("config applied but failed to restart service: %w", err)
			}
			result.Restarted = true
		}
	}

	return result, nil
}

func updateConfig(content string, cfg *calculator.Config) (string, []string) {
	var changes []string
	lines := strings.Split(content, "\n")

	settings := map[string]string{
		"pm":                      string(cfg.PM),
		"pm.max_children":         fmt.Sprintf("%d", cfg.MaxChildren),
		"pm.start_servers":        fmt.Sprintf("%d", cfg.StartServers),
		"pm.min_spare_servers":    fmt.Sprintf("%d", cfg.MinSpareServers),
		"pm.max_spare_servers":    fmt.Sprintf("%d", cfg.MaxSpareServers),
		"pm.max_requests":         fmt.Sprintf("%d", cfg.MaxRequests),
		"pm.process_idle_timeout": cfg.ProcessIdleTimeout,
	}

	// Track which settings we've updated
	updated := make(map[string]bool)

	// Regex patterns for matching settings (handles comments and whitespace)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, ";") {
			continue
		}

		for key, value := range settings {
			// Skip settings not relevant to current PM type
			if !isSettingRelevant(key, cfg.PM) {
				continue
			}

			pattern := regexp.MustCompile(`^;?\s*` + regexp.QuoteMeta(key) + `\s*=`)
			if pattern.MatchString(trimmed) {
				oldLine := lines[i]
				lines[i] = fmt.Sprintf("%s = %s", key, value)
				if oldLine != lines[i] {
					changes = append(changes, fmt.Sprintf("%s: %s -> %s", key, extractValue(oldLine), value))
				}
				updated[key] = true
				break
			}
		}
	}

	// Add any settings that weren't found
	for key, value := range settings {
		if !updated[key] && isSettingRelevant(key, cfg.PM) {
			// Find the [www] section or end of file to add settings
			for i, line := range lines {
				if strings.Contains(line, "[www]") || (i == len(lines)-1) {
					insertIdx := i + 1
					if i == len(lines)-1 {
						insertIdx = i
					}
					newLine := fmt.Sprintf("%s = %s", key, value)
					lines = append(lines[:insertIdx], append([]string{newLine}, lines[insertIdx:]...)...)
					changes = append(changes, fmt.Sprintf("%s: (added) %s", key, value))
					break
				}
			}
		}
	}

	return strings.Join(lines, "\n"), changes
}

func isSettingRelevant(key string, pm calculator.PMType) bool {
	switch key {
	case "pm", "pm.max_children", "pm.max_requests":
		return true
	case "pm.process_idle_timeout":
		return pm == calculator.PMDynamic || pm == calculator.PMOnDemand
	case "pm.start_servers", "pm.min_spare_servers", "pm.max_spare_servers":
		return pm == calculator.PMDynamic
	}
	return false
}

func extractValue(line string) string {
	if strings.HasPrefix(strings.TrimSpace(line), ";") {
		return "(commented out)"
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[1])
	}
	return "(unknown)"
}

func restartService(name string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("sudo", "systemctl", "restart", name)
	case "darwin":
		cmd = exec.Command("brew", "services", "restart", name)
	default:
		return fmt.Errorf("unsupported platform for service restart")
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Confirm prompts the user for confirmation
func Confirm(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// ListConfigFiles returns all detected PHP-FPM config files
func ListConfigFiles() []string {
	var found []string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		}
	}
	return found
}

// ValidateConfigPath checks if path looks like a PHP-FPM config
func ValidateConfigPath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	ext := filepath.Ext(path)
	if ext != ".conf" && ext != "" {
		return fmt.Errorf("file does not appear to be a PHP-FPM config (expected .conf extension): %s", path)
	}

	return nil
}
