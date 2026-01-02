package gitssh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config stores git identity + token needed to register SSH access.
type Config struct {
	Username string
	Email    string
	Token    string
	KeyPath  string
}

// EnsureAccess ensures an SSH key exists locally and is uploaded to GitHub.
func EnsureAccess(cfg Config) error {
	if cfg.Token == "" {
		return errors.New("github token missing in bundle")
	}

	keyPath := cfg.KeyPath
	if keyPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		keyPath = filepath.Join(home, ".ssh", "id_ed25519")
	}

	if err := ensureSSHKey(keyPath); err != nil {
		return err
	}
	if err := uploadKey(cfg.Token, keyPath+".pub"); err != nil {
		return err
	}
	if cfg.Username != "" {
		_ = exec.Command("git", "config", "--global", "user.name", cfg.Username).Run()
	}
	if cfg.Email != "" {
		_ = exec.Command("git", "config", "--global", "user.email", cfg.Email).Run()
	}
	return nil
}

func ensureSSHKey(keyPath string) error {
	if _, err := os.Stat(keyPath); err == nil {
		return nil
	}
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func uploadKey(token, pubPath string) error {
	pub, err := os.ReadFile(pubPath)
	if err != nil {
		return err
	}
	title := fmt.Sprintf("anvil-%s", hostname())
	payload := map[string]string{
		"title": title,
		"key":   strings.TrimSpace(string(pub)),
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://api.github.com/user/keys", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("[git_ssh] Uploaded SSH key to GitHub")
		return nil
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		fmt.Println("[git_ssh] SSH key already registered on GitHub")
		return nil
	}
	return fmt.Errorf("github key upload failed: %s", resp.Status)
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil || h == "" {
		return "anvil-host"
	}
	return h
}
