package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func findGitRoot(dir string) (string, error) {
	for dir != "/" {
		gitDir := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached root
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("no .git directory found")
}

func sandBoxedGit(gitDir string, args ...string) error {
	if gitDir == "" {
		cwd, err := os.Getwd()
		if err != nil { return err }
		gitDir, err = findGitRoot(cwd)
		if err != nil { return err }
	}
	cmd := exec.Command("docker", append([]string{
		"run", "--rm", "-it",
		"--net=host",
		"-v", gitDir + ":" + gitDir,
		"-v", os.Getenv("HOME") + "/.ssh:/root/.ssh",
		"-v", os.Getenv("HOME") + "/.gitconfig:/root/.gitconfig",
		"-v", os.Getenv("HOME") + "/.config/gh:/root/.config/gh",
		"--workdir", os.Getenv("PWD"),
		"--entrypoint", "/usr/bin/git",
		"base:2025-07-22",
	}, args...)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func gitSync(comment, branch string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		os.Exit(1)
	}

	gitDir, err := findGitRoot(cwd)
	if err != nil {
		fmt.Println("No .git directory found")
		os.Exit(1)
	}

	if _, err := os.Stat(filepath.Join(gitDir, ".git")); err == nil {
		if err := os.Chdir(gitDir); err != nil {
			fmt.Println("Error changing directory:", err)
			os.Exit(1)
		}

		commands := [][]string{
			{"add", "."},
			{"commit", "-m", comment},
			{"push", "origin", branch},
		}

		for _, args := range commands {
			if err := sandBoxedGit(gitDir, args...); err != nil {
				fmt.Printf("Error executing git %s: %v\n", strings.Join(args, " "), err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Println("No .git directory found")
		os.Exit(1)
	}
}
