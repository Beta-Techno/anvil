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
