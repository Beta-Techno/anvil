package runtime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

var requirements = []struct {
	cmd string
	pkg string
}{
	{"sops", "sops"},
	{"ansible-playbook", "ansible"},
	{"git", "git"},
}

// CheckPrereqs ensures required binaries exist, installing via apt if possible.
func CheckPrereqs() error {
	var aptUpdated bool
	for _, req := range requirements {
		if _, err := exec.LookPath(req.cmd); err == nil {
			continue
		}

		var installErr error
		if req.cmd == "sops" {
			installErr = installSops(&aptUpdated)
		} else {
			installErr = installViaApt(req.pkg, &aptUpdated)
		}

		if installErr != nil {
			return fmt.Errorf("required command %s not found and automatic install failed: %w", req.cmd, installErr)
		}
	}
	return nil
}

func installViaApt(packageName string, aptUpdated *bool) error {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return fmt.Errorf("apt-get not available")
	}

	runner := buildRunner()
	if runner == nil {
		return fmt.Errorf("neither sudo nor root privileges available")
	}

	if !*aptUpdated {
		if err := runWith(runner, "apt-get", "update"); err != nil {
			return err
		}
		*aptUpdated = true
	}

	return runWith(runner, "apt-get", "install", "-y", packageName)
}

func installSops(aptUpdated *bool) error {
	if err := installViaApt("sops", aptUpdated); err == nil {
		return nil
	}
	return installSopsFromRelease()
}

const sopsVersion = "3.9.4"

func installSopsFromRelease() error {
	arch := runtime.GOARCH
	var suffix string
	switch arch {
	case "amd64":
		suffix = "amd64"
	case "arm64":
		suffix = "arm64"
	default:
		return fmt.Errorf("unsupported architecture %s for sops binary install", arch)
	}

	url := fmt.Sprintf("https://github.com/getsops/sops/releases/download/v%[1]s/sops-v%[1]s.linux.%s", sopsVersion, suffix)
	fmt.Println("[runtime] downloading sops from", url)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "sops-binary-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return err
	}
	tmp.Close()
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}

	target := "/usr/local/bin/sops"
	runner := buildRunner()
	if runner == nil {
		if os.Geteuid() != 0 {
			return fmt.Errorf("need sudo or root to install sops")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.Rename(tmp.Name(), target)
	}

	if err := runCommand("chmod", "+x", tmp.Name()); err != nil {
		return err
	}
	if err := runner("install", "-m", "755", tmp.Name(), target); err != nil {
		return err
	}
	return nil
}

type runnerFunc func(name string, args ...string) error

func buildRunner() runnerFunc {
	if os.Geteuid() == 0 {
		return runCommand
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		return func(name string, args ...string) error {
			return runCommand("sudo", append([]string{name}, args...)...)
		}
	}
	return nil
}

func runWith(rf runnerFunc, name string, args ...string) error {
	return rf(name, args...)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
