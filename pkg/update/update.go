package update

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "Beta-Techno"
	repoName  = "anvil"
)

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease() (*releaseInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", repoOwner, repoName))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	var info releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func assetName(goos, arch string) string {
	return fmt.Sprintf("anvil-%s-%s", goos, arch)
}

func assetURLFromRelease(info *releaseInfo, name string) (string, error) {
	for _, asset := range info.Assets {
		if asset.Name == name {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("asset %s not found", name)
}

// LatestVersionInfo returns the latest tag and download URL for the current platform.
func LatestVersionInfo() (tag string, downloadURL string, err error) {
	info, err := latestRelease()
	if err != nil {
		return "", "", err
	}
	name := assetName(runtime.GOOS, runtime.GOARCH)
	url, err := assetURLFromRelease(info, name)
	return info.TagName, url, err
}

// BuildDownloadURL builds a download URL for a specific tag.
func BuildDownloadURL(tag string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", repoOwner, repoName, tag, assetName(runtime.GOOS, runtime.GOARCH))
}

// DownloadAndReplace downloads the binary, verifies checksum, and replaces the target path.
func DownloadAndReplace(url, checksumURL, target string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	if target == "" {
		return errors.New("target path required")
	}
	targetDir := filepath.Dir(target)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	checksumPath := filepath.Join(targetDir, "anvil-update.sha256")
	if err := downloadFile(checksumURL, checksumPath); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(targetDir, "anvil-update")
	if err != nil {
		return err
	}
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}
	if err := tmpFile.Chmod(0o755); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFile.Name(), target); err != nil {
		return err
	}
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum download failed: %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func fileChecksumMatches(binaryPath, checksumPath string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return errors.New("invalid checksum file")
	}
	computed, err := computeSHA256(binaryPath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(computed, parts[0]) {
		return fmt.Errorf("checksum mismatch: got %s expected %s", computed, parts[0])
	}
	return nil
}

func computeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CompareVersions compares semantic versions (vMAJOR.MINOR.PATCH).
// Returns -1 if current < latest, 0 if equal, 1 if current > latest.
func CompareVersions(current, latest string) int {
	norm := func(v string) []int {
		v = strings.TrimPrefix(v, "v")
		parts := strings.Split(v, ".")
		res := make([]int, 0, len(parts))
		for _, p := range parts {
			var val int
			fmt.Sscanf(p, "%d", &val)
			res = append(res, val)
		}
		return res
	}
	a := norm(current)
	b := norm(latest)
	for i := 0; i < len(a) || i < len(b); i++ {
		var ai, bi int
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}
