package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	name    = "DockerJupyter"
	dimg    = "brew:2025-07-22"
	xauth   = ".Xauthority"
)

func sandboxedJupyter(args ...string) {
	if len(args) > 1 && args[1] == "kill" {
		killContainer()
		return
	}

	if isContainerRunning() {
		fmt.Fprintf(os.Stderr, "Container %s is already running\n", name)
		os.Exit(0)
	}

	runContainer()
}

func killContainer() {
	if !isContainerRunning() {
		fmt.Fprintf(os.Stderr, "Container %s is not running\n", name)
		os.Exit(1)
	}

	cmd := exec.Command("docker", "kill", name)
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to stop container: %v", err)
	}

	fmt.Printf("Stopped container %s\n", name)
	os.Exit(0)
}

func isContainerRunning() bool {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to check running containers: %v", err)
	}

	return strings.Contains(out.String(), name)
}

func runContainer() {
	home := os.Getenv("HOME")

	display := os.Getenv("DISPLAY")
	if display == "" {
		log.Fatal("DISPLAY environment variable not set")
	}

	cmd := exec.Command("docker", "run", "-d", "--rm",
		"--name", name,
		"--net=host",
		"--volume", "/etc/machine-id:/etc/machine-id:ro",
		"--env", "PATH=/usr/bin:/home/linuxbrew/.linuxbrew/bin:/home/linuxbrew/.linuxbrew/sbin",
		"-v", fmt.Sprintf("%s/Jupyter/Notebooks:/root/Notebooks", home),
		"--workdir", "/root/Notebooks",
		"--env", fmt.Sprintf("DISPLAY=%s", display),
		"--volume", "/tmp/.X11-unix:/tmp/.X11-unix",
		"--volume", fmt.Sprintf("%s/Jupyter/Julia:/root/.julia", home),
		"--env", "XAUTHORITY=/root/.Xauthority",
		"--volume", fmt.Sprintf("%s/%s:/root/.Xauthority", home, xauth),
		"--entrypoint", "/usr/bin/jupyter",
		dimg, "lab", "--ServerApp.token=''", "--ServerApp.password=''", "--allow-root", "--NotebookApp.ip=0.0.0.0",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to run container: %v", err)
	}
}
