package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/php-fpm/optimizer/internal/apply"
	"github.com/php-fpm/optimizer/internal/calculator"
	"github.com/php-fpm/optimizer/internal/output"
	"github.com/php-fpm/optimizer/internal/php"
	"github.com/php-fpm/optimizer/internal/system"
)

var version = "dev"

func main() {
	// CLI flags
	var (
		showVersion    bool
		showHelp       bool
		noColor        bool
		onlyConf       bool
		pmType         string
		trafficProfile string
		reservedMemory int
		processMemory  float64
		applyConfig    bool
		configPath     string
		restart        bool
		yes            bool
	)

	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help (shorthand)")
	flag.BoolVar(&noColor, "no-color", false, "Disable colored output")
	flag.BoolVar(&onlyConf, "config-only", false, "Output only the configuration (for piping)")
	flag.BoolVar(&onlyConf, "c", false, "Output only the configuration (shorthand)")
	flag.StringVar(&pmType, "pm", "", "Process manager type: static, dynamic, ondemand (default: auto)")
	flag.StringVar(&trafficProfile, "traffic", "medium", "Traffic profile: low, medium, high")
	flag.IntVar(&reservedMemory, "reserved", 0, "Reserved memory in MB for OS/services (default: auto)")
	flag.Float64Var(&processMemory, "process-mem", 0, "Override PHP process memory in MB (default: auto-detect)")
	flag.BoolVar(&applyConfig, "apply", false, "Apply configuration directly to PHP-FPM config file")
	flag.StringVar(&configPath, "config", "", "Path to PHP-FPM pool config file (default: auto-detect)")
	flag.BoolVar(&restart, "restart", false, "Restart PHP-FPM service after applying (use with --apply)")
	flag.BoolVar(&yes, "yes", false, "Skip confirmation prompts (use with --apply)")
	flag.BoolVar(&yes, "y", false, "Skip confirmation prompts (shorthand)")

	flag.Usage = printUsage
	flag.Parse()

	if showHelp {
		printUsage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Printf("php-fpm-optimizer %s\n", version)
		os.Exit(0)
	}

	// Initialize printer
	printer := output.NewPrinter(os.Stdout, noColor, onlyConf)

	// Print header
	printer.PrintHeader()

	// Detect system info
	sysInfo, err := system.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting system info: %v\n", err)
		os.Exit(1)
	}
	printer.PrintSystemInfo(sysInfo)

	// Detect PHP processes
	phpInfo, err := php.DetectProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not detect PHP processes: %v\n", err)
		phpInfo = &php.ProcessInfo{}
	}
	printer.PrintPHPInfo(phpInfo)

	// Build options
	opts := calculator.DefaultOptions()

	if reservedMemory > 0 {
		opts.ReservedMemoryMB = reservedMemory
	}

	if processMemory > 0 {
		opts.ProcessMemoryMB = processMemory
	}

	switch strings.ToLower(trafficProfile) {
	case "low":
		opts.TrafficProfile = calculator.TrafficLow
	case "high":
		opts.TrafficProfile = calculator.TrafficHigh
	default:
		opts.TrafficProfile = calculator.TrafficMedium
	}

	switch strings.ToLower(pmType) {
	case "static":
		opts.PMType = calculator.PMStatic
	case "dynamic":
		opts.PMType = calculator.PMDynamic
	case "ondemand":
		opts.PMType = calculator.PMOnDemand
	}

	// Calculate configuration
	cfg := calculator.Calculate(sysInfo, phpInfo, opts)

	// Print results
	printer.PrintCalculation(cfg)
	printer.PrintConfig(cfg)
	printer.PrintWarnings(cfg)

	// Apply configuration if requested
	if applyConfig {
		if err := applyConfiguration(cfg, configPath, restart, yes, noColor); err != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
			os.Exit(1)
		}
	} else {
		printer.PrintRecommendations(cfg)
		printer.PrintUsage()
	}
}

func applyConfiguration(cfg *calculator.Config, configPath string, restart, skipConfirm, noColor bool) error {
	// Colors
	green := "\033[32m"
	yellow := "\033[33m"
	cyan := "\033[36m"
	reset := "\033[0m"
	bold := "\033[1m"

	if noColor {
		green, yellow, cyan, reset, bold = "", "", "", "", ""
	}

	fmt.Println()
	fmt.Printf("%s%sApply Configuration%s\n\n", bold, cyan, reset)

	// Find or validate config path
	if configPath == "" {
		var err error
		configPath, err = apply.FindConfigFile()
		if err != nil {
			// List what we searched for
			fmt.Printf("%sSearched locations:%s\n", yellow, reset)
			for _, path := range apply.ListConfigFiles() {
				fmt.Printf("  - %s\n", path)
			}
			return fmt.Errorf("could not auto-detect PHP-FPM config file. Use --config to specify the path")
		}
	} else {
		if err := apply.ValidateConfigPath(configPath); err != nil {
			return err
		}
	}

	fmt.Printf("  Config file:  %s%s%s\n", green, configPath, reset)

	// Show detected service
	serviceName := apply.FindServiceName()
	if serviceName != "" {
		fmt.Printf("  Service:      %s%s%s\n", green, serviceName, reset)
	} else {
		fmt.Printf("  Service:      %s(not detected)%s\n", yellow, reset)
	}

	if restart && serviceName == "" {
		fmt.Printf("\n%sWarning:%s --restart specified but no PHP-FPM service detected\n", yellow, reset)
	}

	fmt.Println()

	// Confirm before proceeding
	if !skipConfirm {
		action := "Apply these settings"
		if restart {
			action += " and restart PHP-FPM"
		}
		if !apply.Confirm(action + "?") {
			fmt.Println("Aborted.")
			return nil
		}
		fmt.Println()
	}

	// Apply the configuration
	result, err := apply.Apply(cfg, configPath, restart)
	if err != nil {
		return err
	}

	// Print results
	fmt.Printf("%s%sChanges Applied%s\n\n", bold, green, reset)

	if len(result.Changes) == 0 {
		fmt.Println("  No changes were necessary (config already up to date)")
	} else {
		for _, change := range result.Changes {
			fmt.Printf("  %s*%s %s\n", cyan, reset, change)
		}
	}

	fmt.Println()
	fmt.Printf("  Backup saved to: %s\n", result.BackupPath)

	if result.Restarted {
		fmt.Printf("  Service %s%s%s restarted successfully\n", green, result.ServiceName, reset)
	} else if restart && result.ServiceName != "" {
		fmt.Printf("\n  %sNote:%s Run 'sudo systemctl restart %s' to apply changes\n", yellow, reset, result.ServiceName)
	} else if !restart {
		fmt.Printf("\n  %sNote:%s Restart PHP-FPM to apply changes:\n", yellow, reset)
		fmt.Println("        sudo systemctl restart php-fpm")
	}

	fmt.Println()

	return nil
}

func printUsage() {
	fmt.Println(`PHP-FPM Process Manager Optimizer

Analyzes your system and calculates optimal PHP-FPM pm configuration.

USAGE:
    php-fpm-optimizer [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -v, --version           Show version
    -c, --config-only       Output only configuration (for piping to file)
    --no-color              Disable colored output

    --pm <type>             Process manager type: static, dynamic, ondemand
                            Default: auto-selected based on traffic profile

    --traffic <profile>     Expected traffic level: low, medium, high
                            Default: medium
                            - low: Uses ondemand PM (saves memory)
                            - medium: Uses dynamic PM (balanced)
                            - high: Uses static PM (fastest response)

    --reserved <MB>         Memory to reserve for OS/services in MB
                            Default: auto-calculated (512MB + 15% of total)

    --process-mem <MB>      Override detected PHP process memory in MB
                            Default: auto-detected from running processes

APPLY OPTIONS:
    --apply                 Apply configuration directly to PHP-FPM config file
    --config <path>         Path to PHP-FPM pool config (default: auto-detect)
    --restart               Restart PHP-FPM service after applying
    -y, --yes               Skip confirmation prompts

EXAMPLES:
    # Auto-detect everything
    php-fpm-optimizer

    # High-traffic production server
    php-fpm-optimizer --traffic high --pm static

    # Low-memory VPS with light traffic
    php-fpm-optimizer --traffic low --reserved 1024

    # Export config directly to file
    php-fpm-optimizer --config-only > /tmp/php-fpm-pm.conf

    # Specify known process memory usage
    php-fpm-optimizer --process-mem 64

    # Apply configuration directly (with confirmation)
    php-fpm-optimizer --apply

    # Apply and restart PHP-FPM (skip confirmation)
    php-fpm-optimizer --apply --restart --yes

    # Apply to specific config file
    php-fpm-optimizer --apply --config /etc/php/8.2/fpm/pool.d/www.conf

QUICK INSTALL:
    curl -fsSL https://example.com/install.sh | sh

For more information, visit: https://github.com/php-fpm/optimizer
`)
}
