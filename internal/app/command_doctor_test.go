package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func TestDoctorReportsHealthyCurrentInstallation(t *testing.T) {
	home := t.TempDir()
	installationPath := filepath.Join(home, "tools", "go", "1.26.5")
	binDir := filepath.Join(installationPath, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "go"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	stateStore := store.New(home)
	state := model.State{
		Defaults:  map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{model.Go: {{Version: "1.26.5", Path: installationPath, Managed: true}}},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.SetCurrent(model.Go, installationPath); err != nil {
		t.Fatal(err)
	}

	currentBin := filepath.Join(home, "tools", "go", "current", "bin")
	t.Setenv("SDK_HOME", home)
	t.Setenv("PATH", currentBin)
	t.Setenv("GOROOT", filepath.Join(home, "tools", "go", "current"))
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"doctor"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, out.String())
	}
	for _, expected := range []string{"[OK]   go current", "[OK]   go executable go", "[OK]   go PATH", "[OK]   GOROOT", "0 error(s)"} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("doctor output does not contain %q:\n%s", expected, out.String())
		}
	}
}

func TestDoctorFailsForBrokenCurrentLink(t *testing.T) {
	home := t.TempDir()
	stateStore := store.New(home)
	installationPath := filepath.Join(home, "tools", "go", "1.26.5")
	if err := os.MkdirAll(installationPath, 0755); err != nil {
		t.Fatal(err)
	}
	state := model.State{
		Defaults:  map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{model.Go: {{Version: "1.26.5", Path: installationPath, Managed: true}}},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	currentPath := stateStore.CurrentPath(model.Go)
	if err := os.MkdirAll(filepath.Dir(currentPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "missing"), currentPath); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SDK_HOME", home)
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"doctor"}, &out, &bytes.Buffer{}); err == nil {
		t.Fatalf("doctor unexpectedly succeeded:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "broken link") {
		t.Fatalf("doctor output does not report broken link:\n%s", out.String())
	}
}
