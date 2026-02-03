package calculator

import (
	"github.com/muuvmuuv/php-tuner/internal/system"
)

// FrankenPHPConfig holds the calculated FrankenPHP configuration
type FrankenPHPConfig struct {
	NumThreads  int
	MaxThreads  int
	WorkerNum   int
	MaxWaitTime string

	// Metadata for display
	ReservedMemoryMB  int
	AvailableMemoryMB int
	ThreadMemoryMB    float64
	Warnings          []string
	Recommendations   []string
}

// FrankenPHPOptions for calculation
type FrankenPHPOptions struct {
	ReservedMemoryMB int            // Memory reserved for OS/other services
	ThreadMemoryMB   float64        // Override detected thread memory
	TrafficProfile   TrafficProfile // Expected traffic level
	WorkerMode       bool           // Using worker mode (long-running)
}

// DefaultFrankenPHPOptions returns sensible defaults
func DefaultFrankenPHPOptions() FrankenPHPOptions {
	return FrankenPHPOptions{
		ReservedMemoryMB: 0, // Auto-calculate
		ThreadMemoryMB:   0, // Auto-detect
		TrafficProfile:   TrafficMedium,
		WorkerMode:       true, // Most FrankenPHP users use worker mode
	}
}

// CalculateFrankenPHP computes optimal FrankenPHP settings
func CalculateFrankenPHP(sysInfo *system.Info, opts FrankenPHPOptions) *FrankenPHPConfig {
	cfg := &FrankenPHPConfig{
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Determine thread memory
	// FrankenPHP threads are lighter than FPM processes since they share memory
	// Default estimate: 30MB per thread (vs ~60MB for FPM)
	cfg.ThreadMemoryMB = opts.ThreadMemoryMB
	if cfg.ThreadMemoryMB == 0 {
		cfg.ThreadMemoryMB = 30 // Conservative default for FrankenPHP
		cfg.Warnings = append(cfg.Warnings,
			"Using estimated 30MB per thread. Use --process-mem to override if known.")
	}

	// Determine reserved memory (for OS, Caddy itself, etc.)
	cfg.ReservedMemoryMB = opts.ReservedMemoryMB
	if cfg.ReservedMemoryMB == 0 {
		// FrankenPHP/Caddy needs less reserved memory than nginx+fpm
		cfg.ReservedMemoryMB = 256 + (sysInfo.MemTotalMB * 10 / 100)
		if cfg.ReservedMemoryMB > 2048 {
			cfg.ReservedMemoryMB = 2048
		}
	}

	// Calculate available memory for PHP threads
	cfg.AvailableMemoryMB = sysInfo.MemTotalMB - cfg.ReservedMemoryMB
	if cfg.AvailableMemoryMB < 128 {
		cfg.AvailableMemoryMB = 128
		cfg.Warnings = append(cfg.Warnings, "Very low available memory, using minimum of 128MB")
	}

	cpuCores := sysInfo.CPUCores

	// Calculate num_threads
	// FrankenPHP default: 2x CPU cores
	// We calculate based on memory available, but cap reasonably
	maxByMemory := int(float64(cfg.AvailableMemoryMB) / cfg.ThreadMemoryMB)
	defaultThreads := cpuCores * 2

	// Use the lower of memory-based or a reasonable CPU-based limit
	cfg.NumThreads = defaultThreads
	if maxByMemory < cfg.NumThreads {
		cfg.NumThreads = maxByMemory
		cfg.Warnings = append(cfg.Warnings,
			"Thread count limited by available memory")
	}

	// Sanity bounds
	if cfg.NumThreads < 2 {
		cfg.NumThreads = 2
	}
	if cfg.NumThreads > 1000 {
		cfg.NumThreads = 1000
		cfg.Warnings = append(cfg.Warnings, "num_threads capped at 1000")
	}

	// max_threads for auto-scaling
	// Allow up to 4x CPU cores or memory limit, whichever is lower
	cfg.MaxThreads = cpuCores * 4
	if cfg.MaxThreads > maxByMemory {
		cfg.MaxThreads = maxByMemory
	}
	if cfg.MaxThreads < cfg.NumThreads {
		cfg.MaxThreads = cfg.NumThreads
	}

	// Worker num (for worker mode)
	// Similar to num_threads but for persistent workers
	if opts.WorkerMode {
		cfg.WorkerNum = cfg.NumThreads
	}

	// max_wait_time based on traffic profile
	switch opts.TrafficProfile {
	case TrafficLow:
		cfg.MaxWaitTime = "" // Disabled for low traffic
	case TrafficHigh:
		cfg.MaxWaitTime = "5s" // Timeout quickly under high load
	default:
		cfg.MaxWaitTime = "10s"
	}

	// Add recommendations
	addFrankenPHPRecommendations(cfg, sysInfo, opts)

	return cfg
}

func addFrankenPHPRecommendations(cfg *FrankenPHPConfig, sysInfo *system.Info, opts FrankenPHPOptions) {
	if opts.WorkerMode {
		cfg.Recommendations = append(cfg.Recommendations,
			"Worker mode keeps your app in memory for faster responses.")
	} else {
		cfg.Recommendations = append(cfg.Recommendations,
			"Consider enabling worker mode for significant performance gains.")
	}

	if sysInfo.MemTotalMB < 1024 {
		cfg.Recommendations = append(cfg.Recommendations,
			"Low memory system detected. Monitor memory usage closely.")
	}

	cfg.Recommendations = append(cfg.Recommendations,
		"FrankenPHP threads share memory, so they're more efficient than FPM processes.")

	cfg.Recommendations = append(cfg.Recommendations,
		"Use the 'watch' directive in development for hot reloading.")
}
