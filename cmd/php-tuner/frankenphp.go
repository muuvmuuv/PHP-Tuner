package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/muuvmuuv/php-tuner/internal/calculator"
	"github.com/muuvmuuv/php-tuner/internal/output"
	"github.com/muuvmuuv/php-tuner/internal/system"
)

func runFrankenPHP(args []string) {
	fs := flag.NewFlagSet("frankenphp", flag.ExitOnError)

	var (
		showHelp       bool
		noColor        bool
		onlyConf       bool
		trafficProfile string
		reservedMemory int
		threadMemory   float64
		workerMode     bool
	)

	fs.BoolVar(&showHelp, "help", false, "Show help message")
	fs.BoolVar(&showHelp, "h", false, "Show help message (shorthand)")
	fs.BoolVar(&noColor, "no-color", false, "Disable colored output")
	fs.BoolVar(&onlyConf, "config-only", false, "Output only the configuration")
	fs.BoolVar(&onlyConf, "c", false, "Output only the configuration (shorthand)")
	fs.StringVar(&trafficProfile, "traffic", "medium", "Traffic profile: low, medium, high")
	fs.IntVar(&reservedMemory, "reserved", 0, "Reserved memory in MB for OS/services")
	fs.Float64Var(&threadMemory, "thread-mem", 0, "Override PHP thread memory in MB")
	fs.BoolVar(&workerMode, "worker", true, "Enable worker mode")

	fs.Usage = func() { printFrankenPHPUsage() }

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if showHelp {
		printFrankenPHPUsage()
		return
	}

	// Initialize printer
	printer := output.NewPrinter(os.Stdout, noColor, onlyConf)

	// Print header
	printer.PrintFrankenPHPHeader()

	// Detect system info
	sysInfo, err := system.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting system info: %v\n", err)
		os.Exit(1)
	}
	printer.PrintSystemInfo(sysInfo)

	// Build options
	opts := calculator.DefaultFrankenPHPOptions()
	opts.WorkerMode = workerMode

	if reservedMemory > 0 {
		opts.ReservedMemoryMB = reservedMemory
	}

	if threadMemory > 0 {
		opts.ThreadMemoryMB = threadMemory
	}

	switch strings.ToLower(trafficProfile) {
	case "low":
		opts.TrafficProfile = calculator.TrafficLow
	case "high":
		opts.TrafficProfile = calculator.TrafficHigh
	default:
		opts.TrafficProfile = calculator.TrafficMedium
	}

	// Calculate configuration
	cfg := calculator.CalculateFrankenPHP(sysInfo, opts)

	// Print results
	printer.PrintFrankenPHPCalculation(cfg)
	printer.PrintFrankenPHPConfig(cfg, workerMode)
	printer.PrintFrankenPHPWarnings(cfg)
	printer.PrintFrankenPHPRecommendations(cfg)
	printer.PrintFrankenPHPUsage()
}

func printFrankenPHPUsage() {
	fmt.Println(`FrankenPHP Optimizer

Analyzes your system and calculates optimal FrankenPHP configuration.

USAGE:
    php-tuner frankenphp [options]
    php-tuner f [options]

OPTIONS:
    -h, --help          Show this help message
    -c, --config-only   Output only configuration (for piping to file)
    --no-color          Disable colored output

    --traffic <level>   Traffic profile: low, medium, high (default: medium)
                        - low: Fewer threads, no wait timeout
                        - medium: Balanced threads
                        - high: More threads, strict timeouts

    --reserved <MB>     Memory to reserve for OS/Caddy in MB
                        Default: auto-calculated (256MB + 10% of total)

    --thread-mem <MB>   Override estimated thread memory in MB
                        Default: 30MB (FrankenPHP threads share memory)

    --worker=false      Disable worker mode (not recommended)

EXAMPLES:
    # Auto-detect everything
    php-tuner frankenphp

    # High-traffic production
    php-tuner f --traffic high

    # Export config to file
    php-tuner f --config-only > Caddyfile.snippet

    # Custom thread memory estimate
    php-tuner f --thread-mem 50

OUTPUT:
    The configuration is output in Caddyfile format, ready to be added
    to your FrankenPHP Caddyfile configuration.`)
}
