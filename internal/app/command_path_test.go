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

func TestPathAndWhichUseCurrentToolLink(t *testing.T) {
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
		Installed: map[model.Tool][]model.InstalledVersion{model.Go: {{Version: "1.26.5", Path: installationPath}}},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.SetCurrent(model.Go, installationPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SDK_HOME", home)

	run := func(args ...string) (string, error) {
		var out bytes.Buffer
		err := Run(context.Background(), args, &out, &bytes.Buffer{})
		return strings.TrimSpace(out.String()), err
	}
	currentPath := filepath.Join(home, "tools", "go", "current")
	if got, err := run("path", "go"); err != nil || got != currentPath {
		t.Fatalf("sdk path go = %q, %v", got, err)
	}
	executable := filepath.Join(currentPath, "bin", "go")
	if got, err := run("which", "go"); err != nil || got != executable {
		t.Fatalf("sdk which go = %q, %v", got, err)
	}
}

func TestPathRejectsToolWithoutDefault(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(home, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SDK_HOME", home)
	var out bytes.Buffer
	err := Run(context.Background(), []string{"path", "go"}, &out, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "no default go version") {
		t.Fatalf("sdk path go error = %v", err)
	}
}
