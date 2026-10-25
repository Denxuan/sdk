package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Denxuan/sdk/internal/model"
)

func TestValidateInstallationRequiresRunnableExecutable(t *testing.T) {
	root := t.TempDir()
	if err := validateInstallation(model.Go, root); err == nil {
		t.Fatal("validateInstallation succeeded without the go executable")
	}

	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	name := executableName(model.Go)
	if err := os.WriteFile(filepath.Join(bin, name), []byte("executable"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := validateInstallation(model.Go, root); err != nil {
		t.Fatalf("validateInstallation rejected runnable executable: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filepath.Join(bin, name), 0644); err != nil {
			t.Fatal(err)
		}
		if err := validateInstallation(model.Go, root); err == nil {
			t.Fatal("validateInstallation accepted a non-executable file")
		}
	}
}
