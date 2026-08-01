package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// NormalizeJavaHome removes the macOS JDK bundle wrapper so that the managed
// version directory itself is a JAVA_HOME directory containing bin/java.
func NormalizeJavaHome(installationPath string) error {
	javaHome := filepath.Join(installationPath, "Contents", "Home")
	info, err := os.Stat(javaHome)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect Java home: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Java home is not a directory: %s", javaHome)
	}

	temporaryHome := installationPath + ".java-home"
	if _, err := os.Lstat(temporaryHome); err == nil {
		return fmt.Errorf("temporary Java home already exists: %s", temporaryHome)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect temporary Java home: %w", err)
	}
	if err := os.Rename(javaHome, temporaryHome); err != nil {
		return fmt.Errorf("move Java home out of bundle: %w", err)
	}
	if err := os.RemoveAll(installationPath); err != nil {
		return fmt.Errorf("remove Java bundle wrapper: %w", err)
	}
	if err := os.Rename(temporaryHome, installationPath); err != nil {
		return fmt.Errorf("place normalized Java home: %w", err)
	}
	return nil
}
