package persona

import (
	"fmt"
	"os"
)

func LoadOverrides(path string) error {
	if path == "" {
		return fmt.Errorf("persona vars path empty")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("persona vars file %s missing: %w", path, err)
	}
	return nil
}
