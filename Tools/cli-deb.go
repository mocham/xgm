package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

var laptopKeepPkgs = []string{
	"iptables", "procps", "iputils-ping", "locales", "uidmap",
	"xinit", "xserver-xorg-input-libinput", "mawk", "tzdata",
	// Networking
	"openssh-server", "rsync", "iproute2",
	// QoL packages
	"ca-certificates", "usr-is-merged", "gpgv", "vim*",
	// Hard dependencies
	"linux-image", "sed", "libpam-systemd", "dash",
	// Essential packages
	"bsdutils", "base-passwd", "bash", "findutils", "grep", "gzip", "hostname",
	"init", "libc-bin", "login", "ncurses-base", "ncurses-bin", "util-linux",
	"diffutils", "e2fsprogs",
}

var vmKeepPkgs = []string {
	// Obsolete: "fonts-dejavu-core", "libxft2", "libxi6", "libxcursor1", 
    "procps", "xinit", "xserver-xorg-input-evdev", "xserver-xorg-core",
    // Networking 
    "openssh-client", "rsync", "iproute2", "ca-certificates", "iputils-ping",
    // QoL packages
    "usr-is-merged", "gpgv", "vim*",
    // Hard dependencies
    "linux-image", "sed", "libpam-systemd", "dash", "mawk",
    // Essential packages
    "bsdutils", "base-passwd", "bash", "findutils", "grep", "gzip", "hostname",
    "init", "libc-bin", "login", "ncurses-base", "ncurses-bin", "util-linux",
}


func printPurgable(keepPkgs []string) {
	// Get installed packages
	installedPkgs, err := getInstalledPackages()
	if err != nil {
		fmt.Printf("Error getting installed packages: %v\n", err)
		return
	}
	var wg sync.WaitGroup

	// Process packages in parallel batches
	chunkSize := 16
	for i := 0; i < len(installedPkgs); i += chunkSize {
		end := i + chunkSize
		if end > len(installedPkgs) {
			end = len(installedPkgs)
		}

		wg.Add(1)
		go func(pkgs []string) {
			defer wg.Done()
			processPackages(keepPkgs, pkgs)
		}(installedPkgs[i:end])
	}

	wg.Wait()
}

func getInstalledPackages() ([]string, error) {
	cmd := exec.Command("/usr/bin/dpkg", "-l")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pkgs []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ii") || strings.HasPrefix(line, "hi") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				pkg := fields[1]
				if pkg != "apt" && pkg != "dpkg" {
					pkgs = append(pkgs, pkg)
				}
			}
		}
	}
	return pkgs, scanner.Err()
}

func processPackages(keepPkgs, pkgs []string) {
	for _, pkg := range pkgs {
		// Skip if in keep list
		if contains(keepPkgs, pkg) {
			continue
		}

		// Check dry-run output
		output, err := dryRunPurge(pkg)
		if err != nil {
			continue
		}

		// Skip if dependency resolution error
		if strings.Contains(output, "pkgProblemResolver") {
			continue
		}

		// Check if any keep packages would be removed
		purgeNeeded := false
		for _, keep := range keepPkgs {
			if strings.Contains(output, keep) {
				purgeNeeded = true
				break
			}
		}

		// Safe to remove
		if !purgeNeeded {
			fmt.Println(pkg)
		}
	}
}

func dryRunPurge(pkg string) (string, error) {
	cmd := exec.Command("apt", "purge", "--dry-run", pkg)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
