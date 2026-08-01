package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Denxuan/sdk/internal/model"
)

func TestInstallUseListAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SDK_HOME", home)
	installPath := filepath.Join(home, "go")
	if err := os.Mkdir(installPath, 0755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) {
		var out bytes.Buffer
		err := Run(context.Background(), args, &out, &bytes.Buffer{})
		return out.String(), err
	}
	if _, err := run("install", "go", "1.26.4", "--path", installPath); err != nil {
		t.Fatal(err)
	}
	if _, err := run("use", "go", "1.26.4"); err != nil {
		t.Fatal(err)
	}
	got, err := run("current", "go")
	if err != nil || strings.TrimSpace(got) != "1.26.4" {
		t.Fatalf("current = %q, %v", got, err)
	}
	got, err = run("list", "go")
	if err != nil || !strings.Contains(got, "* 1.26.4") {
		t.Fatalf("list = %q, %v", got, err)
	}
	if _, err = run("uninstall", "go", "1.26.4"); err == nil {
		t.Fatal("uninstalling the default version succeeded")
	}
}

func TestFormatRemoteVersionsMarksInstalledAndDefault(t *testing.T) {
	var out bytes.Buffer
	state := model.State{
		Defaults: map[model.Tool]string{model.Maven: "3.9.16"},
		Installed: map[model.Tool][]model.InstalledVersion{
			model.Maven: {{Version: "3.9.16"}, {Version: "3.9.15"}},
		},
	}
	formatRemoteVersions(&out, model.Maven, []string{"3.9.16", "3.9.15", "3.9.14"}, state)
	got := out.String()
	for _, expected := range []string{"Available Maven Versions", "> * 3.9.16", "  * 3.9.15", "* - installed", "> - currently in use"} {
		if !strings.Contains(got, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, got)
		}
	}
}
