package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Beta-Techno/anvil-cli/pkg/config"
)

type AnsibleConfig struct {
	RepoPath        string
	VarsFile        string
	PersonaFile     string
	BundleFile      string
	Profile         string
	Tags            string
	Persona         string
	LockboxRepoPath string
	LockboxAgeKey   string
	ManiRepoPath    string
	ManiRepoURL     string
	ManiBin         string
	ManiSyncTags    []string
	ManiRunCommands []string
	ManiManifests   []config.ManifestConfig
	ProfileVars     map[string]any
}

func RunAnsible(cfg AnsibleConfig) error {
	args := []string{"-i", "localhost,", "-c", "local", "playbook.yml", "-e", "@" + cfg.VarsFile, "-e", "@" + cfg.PersonaFile}
	if cfg.BundleFile != "" {
		if _, err := os.Stat(cfg.BundleFile); err == nil {
			args = append(args, "-e", "@"+cfg.BundleFile)
		}
	}
	// Add profile vars as extra-vars
	if len(cfg.ProfileVars) > 0 {
		varsJSON, err := json.Marshal(cfg.ProfileVars)
		if err == nil {
			args = append(args, "-e", string(varsJSON))
		}
	}
	if cfg.Tags != "" && cfg.Tags != "all" {
		args = append(args, "--tags", cfg.Tags)
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = cfg.RepoPath
	cmd.Stdin = os.Stdin

	// Set up progress renderer
	progress, err := NewProgressRenderer()
	if err != nil {
		return fmt.Errorf("failed to create progress renderer: %w", err)
	}
	defer progress.Close()

	// Capture stdout for progress parsing
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	// Capture stderr to log file but also show errors
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	env := os.Environ()
	if cfg.Persona != "" {
		env = append(env, "PERSONA="+cfg.Persona)
	}
	env = append(env, "KEY_BUNDLE_FETCH=skip")
	env = append(env, "ANVIL_REPO_PATH="+cfg.RepoPath)
	if cfg.LockboxRepoPath != "" {
		env = append(env, "LOCKBOX_REPO_PATH="+cfg.LockboxRepoPath)
	}
	if cfg.LockboxAgeKey != "" {
		env = append(env, "LOCKBOX_AGE_KEY_FILE="+cfg.LockboxAgeKey)
	}
	if cfg.ManiRepoPath != "" {
		env = append(env, "MANI_REPO_PATH="+cfg.ManiRepoPath)
	}
	if cfg.ManiRepoURL != "" {
		env = append(env, "MANI_REPO_URL="+cfg.ManiRepoURL)
	}
	if cfg.ManiBin != "" {
		env = append(env, "MANI_BIN="+cfg.ManiBin)
	}
	if len(cfg.ManiSyncTags) > 0 {
		env = append(env, "MANI_SYNC_TAGS="+strings.Join(cfg.ManiSyncTags, ","))
	}
	if len(cfg.ManiRunCommands) > 0 {
		env = append(env, "MANI_RUN_COMMANDS="+strings.Join(cfg.ManiRunCommands, ","))
	}
	if len(cfg.ManiManifests) > 0 {
		if payload, err := json.Marshal(cfg.ManiManifests); err == nil {
			env = append(env, "MANI_MANIFESTS_JSON="+string(payload))
		}
	}

	// Use our custom callback plugin for clean progress output
	env = append(env, "ANSIBLE_STDOUT_CALLBACK=anvil_progress")
	env = append(env, "ANSIBLE_CALLBACK_PLUGINS="+filepath.Join(cfg.RepoPath, "callback_plugins"))

	cmd.Env = env

	fmt.Println()

	if err := cmd.Start(); err != nil {
		return err
	}

	// Process output in goroutines
	done := make(chan bool, 2)

	go func() {
		progress.ProcessOutput(stdout)
		done <- true
	}()

	go func() {
		progress.ProcessOutput(stderr)
		done <- true
	}()

	// Spinner animation
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			progress.Tick()
		}
	}()

	// Wait for output processing
	<-done
	<-done
	ticker.Stop()

	err = cmd.Wait()
	fmt.Println()

	if err != nil || progress.Failed() {
		fmt.Printf("\nSee full log: %s\n", progress.LogPath())
		if progress.Failed() && progress.FailMessage() != "" {
			return fmt.Errorf("provisioning failed")
		}
		return err
	}

	return nil
}

func ResolvePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
