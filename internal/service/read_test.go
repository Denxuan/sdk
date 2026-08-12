package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
)

func TestReadServiceCurrentAndProject(t *testing.T) {
	home := t.TempDir()
	st := store.New(home)
	state := model.State{
		Defaults: map[model.Tool]string{model.Go: "1.26.5"},
		Installed: map[model.Tool][]model.InstalledVersion{
			model.Go: {{Version: "1.26.5", Path: filepath.Join(home, "tools", "go", "1.26.5"), Managed: true}},
		},
	}
	if err := st.Save(state); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(state.Installed[model.Go][0].Path, 0755); err != nil {
		t.Fatal(err)
	}
	projectDir := filepath.Join(home, "project", "nested")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "project", ".sdk-version"), []byte("go=1.26.5\n# comment\n"), 0644); err != nil {
		t.Fatal(err)
	}

	read := NewRead(home)
	current, err := read.CurrentVersions(context.Background())
	if err != nil || len(current) != 1 || current[0].Version != "1.26.5" {
		t.Fatalf("current = %#v, err = %v", current, err)
	}
	project, err := read.ProjectVersions(context.Background(), projectDir)
	if err != nil || !project.Found || project.Versions[model.Go] != "1.26.5" {
		t.Fatalf("project = %#v, err = %v", project, err)
	}
}
