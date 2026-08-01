package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Denxuan/sdk/internal/catalog"
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
	got, err = run("current")
	if err != nil || strings.TrimSpace(got) != "go: 1.26.4" {
		t.Fatalf("all current = %q, %v", got, err)
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
	formatRemoteVersions(&out, model.Maven, []catalog.Version{{Number: "3.9.16"}, {Number: "3.9.15"}, {Number: "3.9.14"}}, state)
	got := out.String()
	for _, expected := range []string{"Available Maven Versions", "> * 3.9.16", "  * 3.9.15", "* - installed", "> - currently in use"} {
		if !strings.Contains(got, expected) {
			t.Errorf("output does not contain %q:\n%s", expected, got)
		}
	}
}

func TestRemoteVersionLabelAddsLTSWithoutChangingVersionLookup(t *testing.T) {
	state := model.State{
		Defaults:  map[model.Tool]string{model.Java: "21.0.12"},
		Installed: map[model.Tool][]model.InstalledVersion{model.Java: {{Version: "21.0.12"}}},
	}
	got := remoteVersionLabel(model.Java, catalog.Version{Number: "21.0.12", LTS: true}, state)
	if got != "> * 21.0.12 LTS" {
		t.Fatalf("label = %q", got)
	}
}

func TestRecommendedVersionPrefersLTS(t *testing.T) {
	versions := []catalog.Version{{Number: "26.0.2"}, {Number: "25.0.4", LTS: true}, {Number: "21.0.12", LTS: true}}
	if got := preferredVersion(versions); got != "25.0.4" {
		t.Fatalf("preferred version = %q", got)
	}
}
