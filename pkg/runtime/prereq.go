package runtime

import (
	"fmt"
	"os"
	"os/exec"
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
		if err := installViaApt(req.pkg, &aptUpdated); err != nil {
			return fmt.Errorf("required command %s not found and automatic install failed: %w", req.cmd, err)
		}
	}
	return nil
}

func installViaApt(packageName string, aptUpdated *bool) error {
	if _, err := exec.LookPath("apt-get"); err != nil {
		return fmt.Errorf("apt-get not available")
	}
	if !*aptUpdated {
		if err := runCommand("sudo", "-n", "apt-get", "update"); err != nil {
			if err := runCommand("apt-get", "update"); err != nil {
				return err
			}
		}
		*aptUpdated = true
	}
	if err := runCommand("sudo", "-n", "apt-get", "install", "-y", packageName); err != nil {
		if err := runCommand("apt-get", "install", "-y", packageName); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
