package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func EnsureRepo(path, url string) error {
	if path == "" {
		return fmt.Errorf("repo path empty")
	}
	if _, err := os.Stat(filepath.Join(path, ".git")); os.IsNotExist(err) {
		fmt.Println("[runtime] Cloning repo", url, "->", path)
		cmd := exec.Command("git", "clone", url, path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	fmt.Println("[runtime] Updating repo at", path)
	cmd := exec.Command("git", "-C", path, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func EnsureVarsFile(repoPath, varsFile string) error {
	if varsFile == "" {
		return fmt.Errorf("vars file path empty")
	}
	if _, err := os.Stat(varsFile); err == nil {
		return nil
	}
	example := filepath.Join(repoPath, "vars", "all.example.yml")
	data, err := os.ReadFile(example)
	if err != nil {
		return fmt.Errorf("failed to read vars template: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(varsFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(varsFile, data, 0o644); err != nil {
		return err
	}
	fmt.Println("[runtime] Created vars file from template:", varsFile)
	return nil
}
