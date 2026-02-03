package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/muuvmuuv/php-tuner/internal/apply"
	"github.com/muuvmuuv/php-tuner/internal/calculator"
	"github.com/muuvmuuv/php-tuner/internal/output"
	"github.com/muuvmuuv/php-tuner/internal/php"
	"github.com/muuvmuuv/php-tuner/internal/system"
)

func runPHPFPM(args []string) {
	fs := flag.NewFlagSet("php-fpm", flag.ExitOnError)

	var (
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

	fs.BoolVar(&showHelp, "help", false, "Show help message")
	fs.BoolVar(&showHelp, "h", false, "Show help message (shorthand)")
	fs.BoolVar(&noColor, "no-color", false, "Disable colored output")
	fs.BoolVar(&onlyConf, "config-only", false, "Output only the configuration")
	fs.BoolVar(&onlyConf, "c", false, "Output only the configuration (shorthand)")
	fs.StringVar(&pmType, "pm", "", "Process manager: static, dynamic, ondemand")
	fs.StringVar(&trafficProfile, "traffic", "medium", "Traffic profile: low, medium, high")
	fs.IntVar(&reservedMemory, "reserved", 0, "Reserved memory in MB for OS/services")
	fs.Float64Var(&processMemory, "process-mem", 0, "Override PHP process memory in MB")
	fs.BoolVar(&applyConfig, "apply", false, "Apply configuration to PHP-FPM config file")
	fs.StringVar(&configPath, "config", "", "Path to PHP-FPM pool config file")
	fs.BoolVar(&restart, "restart", false, "Restart PHP-FPM service after applying")
	fs.BoolVar(&yes, "yes", false, "Skip confirmation prompts")
	fs.BoolVar(&yes, "y", false, "Skip confirmation prompts (shorthand)")

	fs.Usage = func() { printPHPFPMUsage() }

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if showHelp {
		printPHPFPMUsage()
		return
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

func printPHPFPMUsage() {
	fmt.Println(`PHP-FPM Optimizer

Analyzes your system and calculates optimal PHP-FPM process manager configuration.

USAGE:
    php-tuner php-fpm [options]
    php-tuner fpm [options]

OPTIONS:
    -h, --help          Show this help message
    -c, --config-only   Output only configuration (for piping to file)
    --no-color          Disable colored output

    --pm <type>         Process manager type: static, dynamic, ondemand
                        Default: auto-selected based on traffic profile

    --traffic <level>   Traffic profile: low, medium, high (default: medium)
                        - low: Uses ondemand PM (saves memory)
                        - medium: Uses dynamic PM (balanced)
                        - high: Uses static PM (fastest response)

    --reserved <MB>     Memory to reserve for OS/services in MB
                        Default: auto-calculated (512MB + 15% of total)

    --process-mem <MB>  Override detected PHP process memory in MB
                        Default: auto-detected from running processes

APPLY OPTIONS:
    --apply             Apply configuration directly to PHP-FPM config file
    --config <path>     Path to PHP-FPM pool config (default: auto-detect)
    --restart           Restart PHP-FPM service after applying
    -y, --yes           Skip confirmation prompts

EXAMPLES:
    # Auto-detect everything
    php-tuner php-fpm

    # High-traffic production server
    php-tuner fpm --traffic high --pm static

    # Apply configuration directly
    php-tuner fpm --apply --restart --yes

    # Export config to file
    php-tuner fpm --config-only > /tmp/php-fpm.conf

    # Specify known process memory
    php-tuner fpm --process-mem 64

OUTPUT:
    The configuration is output in PHP-FPM pool format, ready to be added
    to your www.conf or custom pool configuration file.`)
}
