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
	if err := Run(context.Background(), []string{"env", "--project"}, &out); err != nil {
		t.Fatalf("sdk env --project failed: %v", err)
	}
	goRootAssignment := "export GOROOT=\"" + projectPath + "\""
	if runtime.GOOS == "windows" {
		goRootAssignment = "$env:GOROOT = '" + projectPath + "'"
	}
	if !strings.Contains(out.String(), goRootAssignment) {
		t.Fatalf("project environment did not use project version:\n%s", out.String())
	}
	globalGoRootAssignment := "export GOROOT=\"" + stateStore.CurrentPath(model.Go) + "\""
	if runtime.GOOS == "windows" {
		globalGoRootAssignment = "$env:GOROOT = '" + stateStore.CurrentPath(model.Go) + "'"
	}
	if strings.Contains(out.String(), globalGoRootAssignment) {
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
	if err := Run(context.Background(), []string{"project", "set", "java", "21.0.12"}, &out); err != nil {
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

func TestProjectInstallSkipsInstalledVersionWithoutChangingDefault(t *testing.T) {
	home := t.TempDir()
	projectVersionPath := filepath.Join(home, "tools", "go", "1.25.0")
	defaultVersionPath := filepath.Join(home, "tools", "go", "1.26.5")
	for _, path := range []string{projectVersionPath, defaultVersionPath} {
		if err := os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	stateStore := store.New(home)
	if err := stateStore.Save(model.State{
		Defaults: map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{model.Go: {
			{Version: "1.25.0", Path: projectVersionPath},
			{Version: "1.26.5", Path: defaultVersionPath},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, projectVersionFile), []byte("go=1.25.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previousDirectory)

	var out bytes.Buffer
	if err := installProject(context.Background(), stateStore, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "go 1.25.0 is already installed") {
		t.Fatalf("unexpected output: %s", out.String())
	}
	state, err := stateStore.Load()
	if err != nil {
		t.Fatal(err)
	}
	if state.Defaults[model.Go] != "1.26.5" {
		t.Fatalf("global Go default changed to %q", state.Defaults[model.Go])
	}
}

func TestProjectStatusReportsMissingAndValidInstallations(t *testing.T) {
	home := t.TempDir()
	goPath := filepath.Join(home, "tools", "go", "1.26.5")
	if err := os.MkdirAll(filepath.Join(goPath, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goPath, "bin", executableName(model.Go)), []byte("go"), 0755); err != nil {
		t.Fatal(err)
	}
	stateStore := store.New(home)
	if err := stateStore.Save(model.State{Installed: map[model.Tool][]model.InstalledVersion{
		model.Go: {{Version: "1.26.5", Path: goPath}},
	}}); err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, projectVersionFile), []byte("go=1.26.5\njava=25.0.5-tem\n"), 0644); err != nil {
		t.Fatal(err)
	}
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previousDirectory)
	t.Setenv("SDK_HOME", home)

	var out bytes.Buffer
	err = Run(context.Background(), []string{"project", "status"}, &out)
	if err == nil {
		t.Fatal("project status succeeded with an uninstalled version")
	}
	for _, expected := range []string{"[OK] go 1.26.5:", "[MISSING] java 25.0.5-tem:", "sdk project install"} {
		if !strings.Contains(out.String(), expected) {
			t.Errorf("project status output missing %q:\n%s", expected, out.String())
		}
	}
}

func TestProjectStatusReportsBrokenExecutable(t *testing.T) {
	home := t.TempDir()
	goPath := filepath.Join(home, "tools", "go", "1.26.5")
	if err := os.MkdirAll(filepath.Join(goPath, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	stateStore := store.New(home)
	if err := stateStore.Save(model.State{Installed: map[model.Tool][]model.InstalledVersion{
		model.Go: {{Version: "1.26.5", Path: goPath}},
	}}); err != nil {
		t.Fatal(err)
	}
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, projectVersionFile), []byte("go=1.26.5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	previousDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previousDirectory)
	t.Setenv("SDK_HOME", home)

	var out bytes.Buffer
	err = Run(context.Background(), []string{"project", "status"}, &out)
	if err == nil || !strings.Contains(out.String(), "[BROKEN] go 1.26.5:") {
		t.Fatalf("project status = %q, %v; want broken executable report", out.String(), err)
	}
}
