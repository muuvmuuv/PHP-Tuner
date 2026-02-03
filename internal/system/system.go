package system

import (
	"bufio"
	"fmt"
	"os"
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
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("unsupported platform: %s (only linux is supported)", runtime.GOOS)
	}

	info := &Info{
		Platform: "linux",
		CPUCores: runtime.NumCPU(),
	}

	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/meminfo: %w", err)
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
		info.MemAvailMB = info.MemFreeMB
	}

	info.MemUsedMB = info.MemTotalMB - info.MemAvailMB

	return info, scanner.Err()
}
