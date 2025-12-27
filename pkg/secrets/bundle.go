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
		fmt.Println("[secrets] Using existing decrypted bundle:", bundlePath)
		return nil
	}
	if err := ensureAgeKey(ageKeyPath); err != nil {
		return err
	}

	tmp, err := os.CreateTemp("", "bundle-*.sops.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	fmt.Println("[secrets] Downloading bundle from", bundleURL)
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
	fmt.Println("[secrets] Bundle decrypted to", bundlePath)
	return nil
}

func ensureAgeKey(path string) error {
	if path == "" {
		return errors.New("age key path is empty")
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if envFile := os.Getenv("SOPS_AGE_KEY_FILE"); envFile != "" {
		data, err := os.ReadFile(envFile)
		if err == nil {
			return writeKey(path, data)
		}
	}
	if env := os.Getenv("SOPS_AGE_KEY"); env != "" {
		return writeKey(path, []byte(env))
	}

	fmt.Println("Enter/paste your age secret key (end with EOF / Ctrl+D):")
	reader := bufio.NewReader(os.Stdin)
	var builder strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			builder.WriteString(line)
			break
		}
		if err != nil {
			return err
		}
		builder.WriteString(line)
	}
	key := strings.TrimSpace(builder.String())
	if key == "" {
		return errors.New("age key input empty")
	}
	return writeKey(path, []byte(key))
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
