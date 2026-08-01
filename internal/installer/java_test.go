package installer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeJavaHomeFlattensMacBundle(t *testing.T) {
	installation := filepath.Join(t.TempDir(), "25.0.4")
	javaBinary := filepath.Join(installation, "Contents", "Home", "bin", "java")
	if err := os.MkdirAll(filepath.Dir(javaBinary), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(javaBinary, []byte("java"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := NormalizeJavaHome(installation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(installation, "bin", "java")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(installation, "Contents")); !os.IsNotExist(err) {
		t.Fatalf("Contents directory still exists: %v", err)
	}
}
