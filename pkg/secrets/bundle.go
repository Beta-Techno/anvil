package secrets

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func Unlock(bundleURL, bundlePath, ageKeyPath string) error {
	if bundlePath == "" {
		return errors.New("bundle path is empty")
	}
	if _, err := os.Stat(bundlePath); err == nil {
		return nil // bundle already decrypted
	}
	if err := ensureAgeKey(ageKeyPath); err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "bundle-*.sops.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	fmt.Println("  Fetching bundle...")
	resp, err := http.Get(bundleURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(bundlePath), 0o700); err != nil {
		return err
	}

	cmd := exec.Command("sops", "decrypt", "--input-type", "yaml", "--output-type", "yaml", tmp.Name())
	cmd.Env = append(os.Environ(), fmt.Sprintf("SOPS_AGE_KEY_FILE=%s", ageKeyPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sops decrypt failed: %w\n%s", err, string(output))
	}
	if err := os.WriteFile(bundlePath, output, 0o600); err != nil {
		return err
	}
	fmt.Println("  Decrypting bundle... ✓")
	return nil
}

func ensureAgeKey(path string) error {
	if path == "" {
		return errors.New("age key path is empty")
	}
	// Already exists at target path
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	// Check SOPS_AGE_KEY_FILE env var
	if envFile := os.Getenv("SOPS_AGE_KEY_FILE"); envFile != "" {
		data, err := os.ReadFile(envFile)
		if err == nil {
			return writeKey(path, data)
		}
	}
	// Check SOPS_AGE_KEY env var (raw key)
	if env := os.Getenv("SOPS_AGE_KEY"); env != "" {
		return writeKey(path, []byte(env))
	}

	// Interactive prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("age key [%s]: ", path)
	input, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	input = strings.TrimSpace(input)

	// Empty input = check if file already exists at default path
	if input == "" {
		return errors.New("age key required")
	}

	// If input looks like a path, read from it
	if !strings.HasPrefix(input, "AGE-SECRET-KEY-") {
		// Treat as file path
		expandedPath := input
		if strings.HasPrefix(input, "~/") {
			if home, err := os.UserHomeDir(); err == nil {
				expandedPath = filepath.Join(home, input[2:])
			}
		}
		data, err := os.ReadFile(expandedPath)
		if err != nil {
			return fmt.Errorf("failed to read age key from %s: %w", expandedPath, err)
		}
		return writeKey(path, data)
	}

	// Input is the raw key
	return writeKey(path, []byte(input))
}

func writeKey(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return nil
}

// SaveKeyFile writes key material to the provided path with 0600 perms.
func SaveKeyFile(path, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errors.New("key input empty")
	}
	return writeKey(path, []byte(key))
}
