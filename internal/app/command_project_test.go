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

func TestProjectEnvironmentOverridesGlobalDefaultFromParentDirectory(t *testing.T) {
	home := t.TempDir()
	globalPath := filepath.Join(home, "tools", "go", "1.26.5")
	projectPath := filepath.Join(home, "tools", "go", "1.25.0")
	for _, path := range []string{globalPath, projectPath} {
		if err := os.MkdirAll(filepath.Join(path, "bin"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	stateStore := store.New(home)
	state := model.State{
		Defaults: map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{
			model.Go: {
				{Version: "1.26.5", Path: globalPath},
				{Version: "1.25.0", Path: projectPath},
			},
		},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := stateStore.SetCurrent(model.Go, globalPath); err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(t.TempDir(), "project")
	childDirectory := filepath.Join(projectRoot, "cmd", "app")
	if err := os.MkdirAll(childDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectRoot, projectVersionFile), []byte("go=1.25.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(childDirectory); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previousDirectory)
	t.Setenv("SDK_HOME", home)

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"env", "--project"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatalf("sdk env --project failed: %v", err)
	}
	if !strings.Contains(out.String(), "export GOROOT=\""+projectPath+"\"") {
		t.Fatalf("project environment did not use project version:\n%s", out.String())
	}
	if strings.Contains(out.String(), "export GOROOT=\""+stateStore.CurrentPath(model.Go)+"\"") {
		t.Fatalf("project environment used global current link:\n%s", out.String())
	}
}

func TestSetProjectVersionCreatesProjectFile(t *testing.T) {
	home := t.TempDir()
	stateStore := store.New(home)
	state := model.State{
		Defaults: map[model.Tool]string{},
		Installed: map[model.Tool][]model.InstalledVersion{
			model.Java: {{Version: "21.0.12", Path: filepath.Join(home, "java")}},
		},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previousDirectory)
	t.Setenv("SDK_HOME", home)

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"project", "set", "java", "21.0.12"}, &out, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(directory, projectVersionFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "java=21.0.12\n" {
		t.Fatalf("project file = %q", contents)
	}
}
