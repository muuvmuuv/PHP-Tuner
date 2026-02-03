package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// Info holds system resource information
type Info struct {
	CPUCores   int
	MemTotalMB int
	MemFreeMB  int
	MemAvailMB int
	MemUsedMB  int
	Platform   string
}

// Detect gathers system information
func Detect() (*Info, error) {
	info := &Info{
		Platform: runtime.GOOS,
	}

	var err error

	switch runtime.GOOS {
	case "linux":
		err = detectLinux(info)
	case "darwin":
		err = detectDarwin(info)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		return nil, err
	}

	return info, nil
}

func detectLinux(info *Info) error {
	// CPU cores
	info.CPUCores = runtime.NumCPU()

	// Memory from /proc/meminfo
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return fmt.Errorf("failed to read /proc/meminfo: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		value, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		// Values in /proc/meminfo are in kB
		valueMB := value / 1024

		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			info.MemTotalMB = valueMB
		case strings.HasPrefix(line, "MemFree:"):
			info.MemFreeMB = valueMB
		case strings.HasPrefix(line, "MemAvailable:"):
			info.MemAvailMB = valueMB
		}
	}

	if info.MemAvailMB == 0 {
		// Fallback for older kernels without MemAvailable
		info.MemAvailMB = info.MemFreeMB
	}

	info.MemUsedMB = info.MemTotalMB - info.MemAvailMB

	return scanner.Err()
}

func detectDarwin(info *Info) error {
	// CPU cores
	info.CPUCores = runtime.NumCPU()

	// Total memory via sysctl
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return fmt.Errorf("failed to get memory size: %w", err)
	}

	memBytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse memory size: %w", err)
	}
	info.MemTotalMB = int(memBytes / 1024 / 1024)

	// Get memory pressure/usage via vm_stat
	out, err = exec.Command("vm_stat").Output()
	if err != nil {
		return fmt.Errorf("failed to get vm_stat: %w", err)
	}

	pageSize := 16384 // Default for Apple Silicon, will try to detect

	// Try to get actual page size
	if psOut, err := exec.Command("sysctl", "-n", "hw.pagesize").Output(); err == nil {
		if ps, err := strconv.Atoi(strings.TrimSpace(string(psOut))); err == nil {
			pageSize = ps
		}
	}

	var freePages, inactivePages int64
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Pages free:") {
			freePages = parseVMStatValue(line)
		} else if strings.Contains(line, "Pages inactive:") {
			inactivePages = parseVMStatValue(line)
		}
	}

	info.MemFreeMB = int((freePages * int64(pageSize)) / 1024 / 1024)
	info.MemAvailMB = int(((freePages + inactivePages) * int64(pageSize)) / 1024 / 1024)
	info.MemUsedMB = info.MemTotalMB - info.MemAvailMB

	return nil
}

func parseVMStatValue(line string) int64 {
	parts := strings.Split(line, ":")
	if len(parts) != 2 {
		return 0
	}
	valStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "."))
	val, _ := strconv.ParseInt(valStr, 10, 64)
	return val
}
