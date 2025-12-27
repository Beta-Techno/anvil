package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type AnsibleConfig struct {
	RepoPath    string
	VarsFile    string
	PersonaFile string
	BundleFile  string
	Profile     string
	Tags        string
	Persona     string
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
	cmd.Env = env

	fmt.Println("[runtime] Running:", cmd.String())
	return cmd.Run()
}

func ResolvePath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}
