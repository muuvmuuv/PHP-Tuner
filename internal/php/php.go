package php

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
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
	info := &ProcessInfo{}

	out, err := exec.Command("sh", "-c", "ps -eo pid,comm | grep -E 'php-fpm|php[0-9]'").Output()
	if err != nil {
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

		if strings.Contains(fields[1], "grep") {
			continue
		}

		memKB := getProcessMemory(pid)
		if memKB > 0 {
			info.Processes = append(info.Processes, Process{
				PID:      pid,
				MemoryKB: memKB,
				Command:  fields[1],
			})
			info.TotalMemMB += float64(memKB) / 1024
		}
	}

	info.ProcessCount = len(info.Processes)
	if info.ProcessCount > 0 {
		info.AvgMemoryMB = info.TotalMemMB / float64(info.ProcessCount)
	}

	return info, nil
}

func getProcessMemory(pid int) int64 {
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
				return val
			}
		}
	}

	return 0
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
		return -1, nil
	}

	multiplier := 1
	if strings.HasSuffix(limit, "G") {
		multiplier = 1024
		limit = strings.TrimSuffix(limit, "G")
	} else if strings.HasSuffix(limit, "M") {
		limit = strings.TrimSuffix(limit, "M")
	} else if strings.HasSuffix(limit, "K") {
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
