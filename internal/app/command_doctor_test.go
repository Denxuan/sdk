package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
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
	if err := os.WriteFile(filepath.Join(binDir, executableName(model.Go)), []byte("#!/bin/sh\n"), 0755); err != nil {
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

	currentPath := stateStore.CurrentPath(model.Go)
	if runtime.GOOS == "windows" {
		currentPath = installationPath
	}
	currentBin := filepath.Join(currentPath, "bin")
	t.Setenv("SDK_HOME", home)
	t.Setenv("PATH", currentBin)
	t.Setenv("GOROOT", currentPath)
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"doctor"}, &out); err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, out.String())
	}
	for _, expected := range []string{"[OK]   go current", "[OK]   go 1.26.5 executable " + executableName(model.Go), "[OK]   go PATH", "[OK]   GOROOT", "0 error(s)"} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("doctor output does not contain %q:\n%s", expected, out.String())
		}
	}
}

func TestDoctorChecksExecutablesForNonDefaultInstallations(t *testing.T) {
	home := t.TempDir()
	currentPath := filepath.Join(home, "tools", "go", "1.26.5")
	brokenPath := filepath.Join(home, "tools", "go", "1.25.0")
	if err := os.MkdirAll(filepath.Join(currentPath, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(currentPath, "bin", executableName(model.Go)), []byte("go"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(brokenPath, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	stateStore := store.New(home)
	if err := stateStore.Save(model.State{
		Defaults: map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{model.Go: {
			{Version: "1.26.5", Path: currentPath},
			{Version: "1.25.0", Path: brokenPath},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.SetCurrent(model.Go, currentPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SDK_HOME", home)
	var out bytes.Buffer
	if err := Run(context.Background(), []string{"doctor"}, &out); err == nil {
		t.Fatalf("doctor ignored broken non-default installation:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "[FAIL] go 1.25.0 executable go") {
		t.Fatalf("doctor did not identify broken non-default version:\n%s", out.String())
	}
}

func TestDoctorFailsForBrokenCurrentLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows selects current versions from SDK state")
	}
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
	if err := Run(context.Background(), []string{"doctor"}, &out); err == nil {
		t.Fatalf("doctor unexpectedly succeeded:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "broken link") {
		t.Fatalf("doctor output does not report broken link:\n%s", out.String())
	}
}
