package secrets

import (
	"fmt"
	"os"
)

// RemoveFileIfExists deletes the given path if it exists, ignoring "not exist" errors.
func RemoveFileIfExists(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

// Reset removes cached bundle + lockbox key files.
func Reset(bundlePath, lockboxKeyPath string) error {
	if err := RemoveFileIfExists(bundlePath); err != nil {
		return fmt.Errorf("remove bundle: %w", err)
	}
	if err := RemoveFileIfExists(lockboxKeyPath); err != nil {
		return fmt.Errorf("remove lockbox key: %w", err)
	}
	return nil
}
