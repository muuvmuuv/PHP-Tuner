package php

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ProcessInfo holds information about PHP-FPM processes
type ProcessInfo struct {
	ProcessCount int
	AvgMemoryMB  float64
	TotalMemMB   float64
	Processes    []Process
}

// Process represents a single PHP-FPM process
type Process struct {
	PID      int
	MemoryKB int64
	Command  string
}

// DetectProcesses finds and analyzes PHP-FPM processes
func DetectProcesses() (*ProcessInfo, error) {
	switch runtime.GOOS {
	case "linux":
		return detectLinux()
	case "darwin":
		return detectDarwin()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func detectLinux() (*ProcessInfo, error) {
	info := &ProcessInfo{}

	// Find PHP-FPM processes
	out, err := exec.Command("sh", "-c", "ps -eo pid,comm | grep -E 'php-fpm|php[0-9]'").Output()
	if err != nil {
		// No processes found is not an error
		return info, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// Skip if this is the grep process itself
		if strings.Contains(fields[1], "grep") {
			continue
		}

		memKB := getProcessMemoryLinux(pid)
		if memKB > 0 {
			proc := Process{
				PID:      pid,
				MemoryKB: memKB,
				Command:  fields[1],
			}
			info.Processes = append(info.Processes, proc)
			info.TotalMemMB += float64(memKB) / 1024
		}
	}

	info.ProcessCount = len(info.Processes)
	if info.ProcessCount > 0 {
		info.AvgMemoryMB = info.TotalMemMB / float64(info.ProcessCount)
	}

	return info, nil
}

func getProcessMemoryLinux(pid int) int64 {
	// Read from /proc/[pid]/status for accurate RSS
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	file, err := os.Open(statusPath)
	if err != nil {
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseInt(fields[1], 10, 64)
				return val // Already in kB
			}
		}
	}

	return 0
}

func detectDarwin() (*ProcessInfo, error) {
	info := &ProcessInfo{}

	// Find PHP-FPM processes with memory info using ps
	out, err := exec.Command("sh", "-c", "ps -eo pid,rss,comm | grep -E 'php-fpm|php[0-9]' | grep -v grep").Output()
	if err != nil {
		// No processes found is not an error
		return info, nil
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		rssKB, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}

		proc := Process{
			PID:      pid,
			MemoryKB: rssKB,
			Command:  fields[2],
		}
		info.Processes = append(info.Processes, proc)
		info.TotalMemMB += float64(rssKB) / 1024
	}

	info.ProcessCount = len(info.Processes)
	if info.ProcessCount > 0 {
		info.AvgMemoryMB = info.TotalMemMB / float64(info.ProcessCount)
	}

	return info, nil
}

// GetPHPMemoryLimit attempts to read the PHP memory_limit setting
func GetPHPMemoryLimit() (int, error) {
	out, err := exec.Command("php", "-r", "echo ini_get('memory_limit');").Output()
	if err != nil {
		return 0, err
	}

	return parseMemoryLimit(strings.TrimSpace(string(out)))
}

func parseMemoryLimit(limit string) (int, error) {
	limit = strings.ToUpper(strings.TrimSpace(limit))

	if limit == "-1" {
		return -1, nil // Unlimited
	}

	multiplier := 1
	if strings.HasSuffix(limit, "G") {
		multiplier = 1024
		limit = strings.TrimSuffix(limit, "G")
	} else if strings.HasSuffix(limit, "M") {
		multiplier = 1
		limit = strings.TrimSuffix(limit, "M")
	} else if strings.HasSuffix(limit, "K") {
		multiplier = 1
		limit = strings.TrimSuffix(limit, "K")
		val, err := strconv.Atoi(limit)
		if err != nil {
			return 0, err
		}
		return val / 1024, nil
	}

	val, err := strconv.Atoi(limit)
	if err != nil {
		return 0, err
	}

	return val * multiplier, nil
}
