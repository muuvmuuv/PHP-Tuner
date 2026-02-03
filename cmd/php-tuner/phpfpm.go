package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

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
	)

	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showHelp, "h", false, "")
	fs.BoolVar(&noColor, "no-color", false, "")
	fs.BoolVar(&onlyConf, "config-only", false, "")
	fs.BoolVar(&onlyConf, "c", false, "")
	fs.StringVar(&pmType, "pm", "", "")
	fs.StringVar(&trafficProfile, "traffic", "medium", "")
	fs.IntVar(&reservedMemory, "reserved", 0, "")
	fs.Float64Var(&processMemory, "process-mem", 0, "")

	fs.Usage = func() { printPHPFPMUsage() }

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if showHelp {
		printPHPFPMUsage()
		return
	}

	printer := output.NewPrinter(os.Stdout, noColor, onlyConf)
	printer.PrintHeader()

	sysInfo, err := system.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error detecting system info: %v\n", err)
		os.Exit(1)
	}
	printer.PrintSystemInfo(sysInfo)

	phpInfo, err := php.DetectProcesses()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not detect PHP processes: %v\n", err)
		phpInfo = &php.ProcessInfo{}
	}
	printer.PrintPHPInfo(phpInfo)

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

	cfg := calculator.Calculate(sysInfo, phpInfo, opts)

	printer.PrintCalculation(cfg)
	printer.PrintConfig(cfg)
	printer.PrintWarnings(cfg)
	printer.PrintRecommendations(cfg)
	printer.PrintUsage()
}

func printPHPFPMUsage() {
	fmt.Println(`PHP-FPM Optimizer

USAGE:
    php-tuner php-fpm [options]
    php-tuner fpm [options]

OPTIONS:
    -h, --help          Show help
    -c, --config-only   Output only configuration
    --no-color          Disable colors
    --pm <type>         static, dynamic, ondemand (default: auto)
    --traffic <level>   low, medium, high (default: medium)
    --reserved <MB>     Reserved memory for OS/services
    --process-mem <MB>  Override PHP process memory

EXAMPLES:
    php-tuner fpm
    php-tuner fpm --traffic high --pm static
    php-tuner fpm -c > www.conf`)
}
