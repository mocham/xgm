package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

type Process struct {
	PID      string
	User     string
	RSS      float64
	Command  string
	Elapsed  string
	CPUTime  string
	SortKey  float64
}

func printMeminfo(args ...string) {
	if len(args) > 1 {
		switch args[1] {
		case "-v":
			n := 20
			if len(args) > 2 {
				var err error
				n, err = strconv.Atoi(args[2])
				if err != nil {
					n = 20
				}
			}
			processes := getProcesses("-rss")
			printProcesses(processes, n)
		case "-t":
			n := 20
			if len(args) > 2 {
				var err error
				n, err = strconv.Atoi(args[2])
				if err != nil {
					n = 20
				}
			}
			processes := getProcesses("-time")
			printProcesses(processes, n)
		default:
			printMemoryInfo()
		}
	} else {
		printMemoryInfo()
	}
}

func getProcesses(sortBy string) []Process {
	cmd := exec.Command("ps", "-eo", "pid,user,rss,comm,etime,time", "--sort", sortBy)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing ps command:", err)
		os.Exit(1)
	}

	lines := strings.Split(string(output), "\n")
	var processes []Process

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		rssKB, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			continue
		}

		process := Process{
			PID:     fields[0],
			User:    fields[1],
			RSS:     rssKB / 1024,
			Command: fields[3],
			Elapsed: formatElapsedTime(fields[4]),
			CPUTime: formatCPUTime(fields[5]),
		}

		if sortBy == "-rss" {
			process.SortKey = rssKB
		} else {
			process.SortKey = parseCPUTime(fields[5])
		}

		processes = append(processes, process)
	}

	if sortBy == "-rss" {
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].SortKey > processes[j].SortKey
		})
	} else {
		sort.Slice(processes, func(i, j int) bool {
			return processes[i].SortKey > processes[j].SortKey
		})
	}

	return processes
}

func formatElapsedTime(etime string) string {
	if strings.Contains(etime, "-") {
		return etime
	}
	return etime
}

func formatCPUTime(cputime string) string {
	parts := strings.Split(cputime, ":")
	switch len(parts) {
	case 2:
		return "00:" + cputime
	case 3:
		return cputime
	case 4:
		dayParts := strings.Split(parts[0], "-")
		if len(dayParts) > 1 {
			return dayParts[0] + "-" + parts[1] + ":" + parts[2] + ":" + parts[3]
		}
		return cputime
	default:
		return cputime
	}
}

func parseCPUTime(cputime string) float64 {
	total := 0.0
	parts := strings.Split(cputime, ":")

	// Handle days if present
	if strings.Contains(parts[0], "-") {
		dayParts := strings.Split(parts[0], "-")
		days, _ := strconv.ParseFloat(dayParts[0], 64)
		total += days * 86400
		parts[0] = dayParts[1]
	}

	// Reverse the remaining parts to process from seconds up
	for i := len(parts) - 1; i >= 0; i-- {
		val, _ := strconv.ParseFloat(parts[i], 64)
		switch len(parts) - 1 - i {
		case 0: // seconds
			total += val
		case 1: // minutes
			total += val * 60
		case 2: // hours
			total += val * 3600
		}
	}

	return total
}

func printProcesses(processes []Process, n int) {
	if n > len(processes) {
		n = len(processes)
	}

	fmt.Printf("%-6s %-8s %-10s %-12s %-12s %s\n", "PID", "USER", "MEM(MB)", "ELAPSED", "CPU_TIME", "COMMAND")
	for i := 0; i < n; i++ {
		p := processes[i]
		fmt.Printf("%-6s %-8s %-10.1f %-12s %-12s %s\n", p.PID, p.User, p.RSS, p.Elapsed, p.CPUTime, p.Command)
	}
}

func printMemoryInfo() {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		fmt.Println("Error reading /proc/meminfo:", err)
		os.Exit(1)
	}

	lines := strings.Split(string(data), "\n")
	var total, avail float64

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total, _ = strconv.ParseFloat(fields[1], 64)
		case "MemAvailable:":
			avail, _ = strconv.ParseFloat(fields[1], 64)
		}
	}

	if total == 0 {
		fmt.Println("Could not read memory information")
		return
	}

	used := total - avail
	fmt.Printf("Total: %.2f GB\n", total/1024/1024)
	fmt.Printf("Used: %.2f GB (%.1f%%)\n", used/1024/1024, (used/total)*100)
	fmt.Printf("Free: %.2f GB\n", avail/1024/1024)
}
