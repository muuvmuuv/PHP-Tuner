package calculator

import (
	"math"

	"github.com/muuvmuuv/php-tuner/internal/php"
	"github.com/muuvmuuv/php-tuner/internal/system"
)

// PMType represents the process manager type
type PMType string

const (
	PMStatic   PMType = "static"
	PMDynamic  PMType = "dynamic"
	PMOnDemand PMType = "ondemand"
)

// TrafficProfile represents expected traffic patterns
type TrafficProfile string

const (
	TrafficLow    TrafficProfile = "low"
	TrafficMedium TrafficProfile = "medium"
	TrafficHigh   TrafficProfile = "high"
)

// Config holds the calculated PHP-FPM configuration
type Config struct {
	PM                 PMType
	MaxChildren        int
	StartServers       int
	MinSpareServers    int
	MaxSpareServers    int
	MaxRequests        int
	ProcessIdleTimeout string

	// Metadata for display
	ReservedMemoryMB  int
	AvailableMemoryMB int
	ProcessMemoryMB   float64
	Warnings          []string
	Recommendations   []string
}

// Options for calculation
type Options struct {
	ReservedMemoryMB int            // Memory reserved for OS/other services
	ProcessMemoryMB  float64        // Override detected process memory
	TrafficProfile   TrafficProfile // Expected traffic level
	PMType           PMType         // Desired PM type (empty = auto)
}

// DefaultOptions returns sensible defaults
func DefaultOptions() Options {
	return Options{
		ReservedMemoryMB: 0, // Auto-calculate
		ProcessMemoryMB:  0, // Auto-detect
		TrafficProfile:   TrafficMedium,
		PMType:           "", // Auto-select
	}
}

// Calculate computes optimal PHP-FPM settings
func Calculate(sysInfo *system.Info, phpInfo *php.ProcessInfo, opts Options) *Config {
	cfg := &Config{
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Determine process memory
	cfg.ProcessMemoryMB = determineProcessMemory(phpInfo, opts)

	// Determine reserved memory (for OS, DB, web server, etc.)
	cfg.ReservedMemoryMB = determineReservedMemory(sysInfo, opts)

	// Calculate available memory for PHP-FPM
	cfg.AvailableMemoryMB = sysInfo.MemTotalMB - cfg.ReservedMemoryMB
	if cfg.AvailableMemoryMB < 256 {
		cfg.AvailableMemoryMB = 256
		cfg.Warnings = append(cfg.Warnings, "Very low available memory, using minimum of 256MB")
	}

	// Determine PM type
	cfg.PM = determinePMType(opts, sysInfo)

	// Calculate max_children based on available memory and process size
	if cfg.ProcessMemoryMB > 0 {
		cfg.MaxChildren = int(math.Floor(float64(cfg.AvailableMemoryMB) / cfg.ProcessMemoryMB))
	} else {
		// Fallback: estimate based on memory_limit or default
		cfg.MaxChildren = int(math.Floor(float64(cfg.AvailableMemoryMB) / 64)) // Assume 64MB default
		cfg.Warnings = append(cfg.Warnings, "Could not detect PHP process memory, using 64MB estimate")
	}

	// Apply sanity bounds
	if cfg.MaxChildren < 5 {
		cfg.MaxChildren = 5
		cfg.Warnings = append(cfg.Warnings, "max_children increased to minimum of 5")
	}
	if cfg.MaxChildren > 1000 {
		cfg.MaxChildren = 1000
		cfg.Warnings = append(cfg.Warnings, "max_children capped at 1000")
	}

	// Calculate other settings based on CPU cores
	cpuCores := sysInfo.CPUCores

	cfg.StartServers = cpuCores * 4
	cfg.MinSpareServers = cpuCores * 2
	cfg.MaxSpareServers = cpuCores * 4

	// Ensure spare servers don't exceed max_children
	if cfg.StartServers > cfg.MaxChildren {
		cfg.StartServers = cfg.MaxChildren
	}
	if cfg.MinSpareServers > cfg.MaxChildren {
		cfg.MinSpareServers = cfg.MaxChildren / 2
	}
	if cfg.MaxSpareServers > cfg.MaxChildren {
		cfg.MaxSpareServers = cfg.MaxChildren
	}

	// Ensure min <= start <= max for spare servers
	if cfg.MinSpareServers > cfg.StartServers {
		cfg.MinSpareServers = cfg.StartServers
	}
	if cfg.MaxSpareServers < cfg.StartServers {
		cfg.MaxSpareServers = cfg.StartServers
	}

	// Set idle timeout based on traffic profile
	switch opts.TrafficProfile {
	case TrafficLow:
		cfg.ProcessIdleTimeout = "10s"
	case TrafficHigh:
		cfg.ProcessIdleTimeout = "3s"
	default:
		cfg.ProcessIdleTimeout = "5s"
	}

	// max_requests helps prevent memory leaks
	cfg.MaxRequests = 500

	// Add recommendations
	addRecommendations(cfg, sysInfo, opts)

	return cfg
}

func determineProcessMemory(phpInfo *php.ProcessInfo, opts Options) float64 {
	if opts.ProcessMemoryMB > 0 {
		return opts.ProcessMemoryMB
	}

	if phpInfo != nil && phpInfo.AvgMemoryMB > 0 {
		return phpInfo.AvgMemoryMB
	}

	// Try to get memory_limit as upper bound estimate
	if limit, err := php.GetPHPMemoryLimit(); err == nil && limit > 0 {
		// Use 50% of memory_limit as estimate (processes rarely use full limit)
		return float64(limit) / 2
	}

	return 0 // Will trigger fallback
}

func determineReservedMemory(sysInfo *system.Info, opts Options) int {
	if opts.ReservedMemoryMB > 0 {
		return opts.ReservedMemoryMB
	}

	// Auto-calculate: reserve memory for OS and other services
	// Base: 512MB minimum for OS
	// Plus: 15% of total memory for buffers/cache/other services
	reserved := 512 + (sysInfo.MemTotalMB * 15 / 100)

	// Cap at 4GB for very large memory systems
	if reserved > 4096 {
		reserved = 4096
	}

	return reserved
}

func determinePMType(opts Options, sysInfo *system.Info) PMType {
	if opts.PMType != "" {
		return opts.PMType
	}

	// Auto-select based on traffic profile and resources
	switch opts.TrafficProfile {
	case TrafficLow:
		return PMOnDemand
	case TrafficHigh:
		return PMStatic
	default:
		// Medium traffic: use dynamic
		return PMDynamic
	}
}

func addRecommendations(cfg *Config, sysInfo *system.Info, opts Options) {
	if cfg.PM == PMStatic {
		cfg.Recommendations = append(cfg.Recommendations,
			"Static PM keeps all workers running. Best for high-traffic, dedicated PHP servers.")
	}

	if cfg.PM == PMOnDemand {
		cfg.Recommendations = append(cfg.Recommendations,
			"Ondemand PM spawns workers only when needed. Best for low-traffic or shared hosting.")
	}

	if cfg.PM == PMDynamic {
		cfg.Recommendations = append(cfg.Recommendations,
			"Dynamic PM balances memory usage and response time. Good for most use cases.")
	}

	if sysInfo.MemTotalMB < 2048 {
		cfg.Recommendations = append(cfg.Recommendations,
			"Consider using 'ondemand' PM on low-memory systems to conserve resources.")
	}

	if cfg.MaxChildren > 100 {
		cfg.Recommendations = append(cfg.Recommendations,
			"High max_children value. Monitor for diminishing returns due to context switching.")
	}

	cfg.Recommendations = append(cfg.Recommendations,
		"Set pm.max_requests to prevent memory leaks from accumulating over time.")

	cfg.Recommendations = append(cfg.Recommendations,
		"Consider separate pools for frontend/backend with different PM configurations.")
}
