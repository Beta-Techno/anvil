package runtime

import (
	"errors"
	"fmt"
	"os/exec"
)

func CheckPrereqs() error {
	bins := []string{"sops", "ansible-playbook", "git"}
	for _, bin := range bins {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("required command %s not found", bin)
		}
	}
	return nil
}

func EnsurePersonaFile(path string) error {
	if path == "" {
		return errors.New("persona file path empty")
	}
	if _, err := exec.LookPath("ansible-playbook"); err != nil {
		return err
	}
	return nil
}
