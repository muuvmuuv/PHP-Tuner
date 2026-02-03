package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/muuvmuuv/php-tuner/internal/calculator"
	"github.com/muuvmuuv/php-tuner/internal/php"
	"github.com/muuvmuuv/php-tuner/internal/system"
)

// Colors for terminal output
const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
)

// Printer handles output formatting
type Printer struct {
	w        io.Writer
	noColor  bool
	onlyConf bool
}

// NewPrinter creates a new output printer
func NewPrinter(w io.Writer, noColor, onlyConf bool) *Printer {
	return &Printer{w: w, noColor: noColor, onlyConf: onlyConf}
}

func (p *Printer) color(c, text string) string {
	if p.noColor {
		return text
	}
	return c + text + Reset
}

// PrintHeader prints the program header
func (p *Printer) PrintHeader() {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color(Bold+Cyan, "PHP-FPM Process Manager Optimizer"))
	fmt.Fprintln(p.w, p.color(Dim, strings.Repeat("─", 40)))
	fmt.Fprintln(p.w)
}

// PrintSystemInfo displays detected system information
func (p *Printer) PrintSystemInfo(info *system.Info) {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "System Information"))
	fmt.Fprintln(p.w)

	p.printRow("Platform", info.Platform)
	p.printRow("CPU Cores", fmt.Sprintf("%d", info.CPUCores))
	p.printRow("Total Memory", fmt.Sprintf("%d MB", info.MemTotalMB))
	p.printRow("Available Memory", fmt.Sprintf("%d MB", info.MemAvailMB))
	p.printRow("Used Memory", fmt.Sprintf("%d MB", info.MemUsedMB))
	fmt.Fprintln(p.w)
}

// PrintPHPInfo displays detected PHP process information
func (p *Printer) PrintPHPInfo(info *php.ProcessInfo) {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "PHP-FPM Processes"))
	fmt.Fprintln(p.w)

	if info.ProcessCount == 0 {
		fmt.Fprintln(p.w, p.color(Yellow, "  No PHP-FPM processes detected"))
		fmt.Fprintln(p.w, p.color(Dim, "  Using estimates based on php.ini memory_limit"))
	} else {
		p.printRow("Process Count", fmt.Sprintf("%d", info.ProcessCount))
		p.printRow("Average Memory", fmt.Sprintf("%.1f MB", info.AvgMemoryMB))
		p.printRow("Total Memory", fmt.Sprintf("%.1f MB", info.TotalMemMB))
	}
	fmt.Fprintln(p.w)
}

// PrintCalculation displays the calculation summary
func (p *Printer) PrintCalculation(cfg *calculator.Config) {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "Calculation"))
	fmt.Fprintln(p.w)

	p.printRow("Reserved Memory", fmt.Sprintf("%d MB (for OS/services)", cfg.ReservedMemoryMB))
	p.printRow("Available for PHP", fmt.Sprintf("%d MB", cfg.AvailableMemoryMB))
	p.printRow("Process Memory", fmt.Sprintf("%.1f MB", cfg.ProcessMemoryMB))
	p.printRow("Formula", fmt.Sprintf("%d MB / %.1f MB = %d workers",
		cfg.AvailableMemoryMB, cfg.ProcessMemoryMB, cfg.MaxChildren))
	fmt.Fprintln(p.w)
}

// PrintConfig displays the recommended configuration
func (p *Printer) PrintConfig(cfg *calculator.Config) {
	if !p.onlyConf {
		fmt.Fprintln(p.w, p.color(Bold+Green, "Recommended Configuration"))
		fmt.Fprintln(p.w)
	}

	// Always print config (even in onlyConf mode)
	fmt.Fprintf(p.w, "pm = %s\n", cfg.PM)
	fmt.Fprintf(p.w, "pm.max_children = %d\n", cfg.MaxChildren)

	if cfg.PM == calculator.PMDynamic || cfg.PM == calculator.PMOnDemand {
		fmt.Fprintf(p.w, "pm.process_idle_timeout = %s\n", cfg.ProcessIdleTimeout)
	}

	if cfg.PM == calculator.PMDynamic {
		fmt.Fprintf(p.w, "pm.start_servers = %d\n", cfg.StartServers)
		fmt.Fprintf(p.w, "pm.min_spare_servers = %d\n", cfg.MinSpareServers)
		fmt.Fprintf(p.w, "pm.max_spare_servers = %d\n", cfg.MaxSpareServers)
	}

	fmt.Fprintf(p.w, "pm.max_requests = %d\n", cfg.MaxRequests)

	if !p.onlyConf {
		fmt.Fprintln(p.w)
	}
}

// PrintWarnings displays any warnings
func (p *Printer) PrintWarnings(cfg *calculator.Config) {
	if p.onlyConf || len(cfg.Warnings) == 0 {
		return
	}

	fmt.Fprintln(p.w, p.color(Bold+Yellow, "Warnings"))
	fmt.Fprintln(p.w)
	for _, w := range cfg.Warnings {
		fmt.Fprintf(p.w, "  %s %s\n", p.color(Yellow, "!"), w)
	}
	fmt.Fprintln(p.w)
}

// PrintRecommendations displays recommendations
func (p *Printer) PrintRecommendations(cfg *calculator.Config) {
	if p.onlyConf || len(cfg.Recommendations) == 0 {
		return
	}

	fmt.Fprintln(p.w, p.color(Bold+Blue, "Recommendations"))
	fmt.Fprintln(p.w)
	for _, r := range cfg.Recommendations {
		fmt.Fprintf(p.w, "  %s %s\n", p.color(Cyan, "*"), r)
	}
	fmt.Fprintln(p.w)
}

// PrintUsage displays how to apply the configuration
func (p *Printer) PrintUsage() {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "How to Apply"))
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "  1. Edit your PHP-FPM pool configuration:")
	fmt.Fprintln(p.w, p.color(Dim, "     /etc/php/8.x/fpm/pool.d/www.conf"))
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "  2. Restart PHP-FPM:")
	fmt.Fprintln(p.w, p.color(Dim, "     sudo systemctl restart php-fpm"))
	fmt.Fprintln(p.w)
}

// PrintFrankenPHPHeader prints the FrankenPHP header
func (p *Printer) PrintFrankenPHPHeader() {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, p.color(Bold+Cyan, "FrankenPHP Optimizer"))
	fmt.Fprintln(p.w, p.color(Dim, strings.Repeat("─", 40)))
	fmt.Fprintln(p.w)
}

// PrintFrankenPHPCalculation displays the FrankenPHP calculation summary
func (p *Printer) PrintFrankenPHPCalculation(cfg *calculator.FrankenPHPConfig) {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "Calculation"))
	fmt.Fprintln(p.w)

	p.printRow("Reserved Memory", fmt.Sprintf("%d MB (for OS/Caddy)", cfg.ReservedMemoryMB))
	p.printRow("Available for PHP", fmt.Sprintf("%d MB", cfg.AvailableMemoryMB))
	p.printRow("Thread Memory", fmt.Sprintf("%.1f MB", cfg.ThreadMemoryMB))
	p.printRow("Formula", fmt.Sprintf("%d MB / %.1f MB = %d threads",
		cfg.AvailableMemoryMB, cfg.ThreadMemoryMB, cfg.NumThreads))
	fmt.Fprintln(p.w)
}

// PrintFrankenPHPConfig displays the recommended FrankenPHP configuration
func (p *Printer) PrintFrankenPHPConfig(cfg *calculator.FrankenPHPConfig, workerMode bool) {
	if !p.onlyConf {
		fmt.Fprintln(p.w, p.color(Bold+Green, "Recommended Configuration"))
		fmt.Fprintln(p.w)
	}

	// Print Caddyfile format
	fmt.Fprintln(p.w, "{")
	fmt.Fprintln(p.w, "    frankenphp {")
	fmt.Fprintf(p.w, "        num_threads %d\n", cfg.NumThreads)

	if cfg.MaxThreads > cfg.NumThreads {
		fmt.Fprintf(p.w, "        max_threads %d\n", cfg.MaxThreads)
	}

	if cfg.MaxWaitTime != "" {
		fmt.Fprintf(p.w, "        max_wait_time %s\n", cfg.MaxWaitTime)
	}

	if workerMode && cfg.WorkerNum > 0 {
		fmt.Fprintln(p.w, "        worker {")
		fmt.Fprintln(p.w, "            file /path/to/your/public/index.php")
		fmt.Fprintf(p.w, "            num %d\n", cfg.WorkerNum)
		fmt.Fprintln(p.w, "        }")
	}

	fmt.Fprintln(p.w, "    }")
	fmt.Fprintln(p.w, "}")

	if !p.onlyConf {
		fmt.Fprintln(p.w)
	}
}

// PrintFrankenPHPWarnings displays FrankenPHP warnings
func (p *Printer) PrintFrankenPHPWarnings(cfg *calculator.FrankenPHPConfig) {
	if p.onlyConf || len(cfg.Warnings) == 0 {
		return
	}

	fmt.Fprintln(p.w, p.color(Bold+Yellow, "Warnings"))
	fmt.Fprintln(p.w)
	for _, w := range cfg.Warnings {
		fmt.Fprintf(p.w, "  %s %s\n", p.color(Yellow, "!"), w)
	}
	fmt.Fprintln(p.w)
}

// PrintFrankenPHPRecommendations displays FrankenPHP recommendations
func (p *Printer) PrintFrankenPHPRecommendations(cfg *calculator.FrankenPHPConfig) {
	if p.onlyConf || len(cfg.Recommendations) == 0 {
		return
	}

	fmt.Fprintln(p.w, p.color(Bold+Blue, "Recommendations"))
	fmt.Fprintln(p.w)
	for _, r := range cfg.Recommendations {
		fmt.Fprintf(p.w, "  %s %s\n", p.color(Cyan, "*"), r)
	}
	fmt.Fprintln(p.w)
}

// PrintFrankenPHPUsage displays how to apply FrankenPHP configuration
func (p *Printer) PrintFrankenPHPUsage() {
	if p.onlyConf {
		return
	}
	fmt.Fprintln(p.w, p.color(Bold, "How to Apply"))
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "  1. Add the configuration to your Caddyfile:")
	fmt.Fprintln(p.w, p.color(Dim, "     /etc/frankenphp/Caddyfile"))
	fmt.Fprintln(p.w, p.color(Dim, "     or ./Caddyfile (current directory)"))
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "  2. Restart FrankenPHP:")
	fmt.Fprintln(p.w, p.color(Dim, "     frankenphp reload"))
	fmt.Fprintln(p.w, p.color(Dim, "     # or with Docker:"))
	fmt.Fprintln(p.w, p.color(Dim, "     docker compose restart"))
	fmt.Fprintln(p.w)
}

func (p *Printer) printRow(label, value string) {
	fmt.Fprintf(p.w, "  %-20s %s\n", p.color(Dim, label), value)
}
