package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
}

func RunAnsible(cfg AnsibleConfig) error {
	args := []string{"-i", "localhost,", "-c", "local", "playbook.yml", "-e", "@" + cfg.VarsFile, "-e", "@" + cfg.PersonaFile}
	if cfg.BundleFile != "" {
		if _, err := os.Stat(cfg.BundleFile); err == nil {
			args = append(args, "-e", "@"+cfg.BundleFile)
		}
	}
	if cfg.Tags != "" && cfg.Tags != "all" {
		args = append(args, "--tags", cfg.Tags)
	}

	cmd := exec.Command("ansible-playbook", args...)
	cmd.Dir = cfg.RepoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

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
	cmd.Env = env

	fmt.Println("  Provisioning...")
	fmt.Println()
	return cmd.Run()
}

func ResolvePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
