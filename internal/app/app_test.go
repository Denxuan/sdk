package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Denxuan/sdk/internal/catalog"
	"github.com/Denxuan/sdk/internal/model"
	"github.com/Denxuan/sdk/internal/store"
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
	currentLink := filepath.Join(home, "tools", "go", "current")
	target, err := os.Readlink(currentLink)
	if err != nil || target != installPath {
		t.Fatalf("current link = %q, %v", target, err)
	}
	environment, err := run("env")
	if err != nil || !strings.Contains(environment, "GOROOT=\""+currentLink+"\"") {
		t.Fatalf("environment = %q, %v", environment, err)
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

func TestIsAffirmative(t *testing.T) {
	for _, answer := range []string{"y", "Y", "yes", " YES "} {
		if !isAffirmative(answer) {
			t.Errorf("%q was not accepted", answer)
		}
	}
	for _, answer := range []string{"", "n", "no", "anything else"} {
		if isAffirmative(answer) {
			t.Errorf("%q was accepted", answer)
		}
	}
}

func TestReplaceInitializationBlockIsIdempotent(t *testing.T) {
	block := zshInitialization("/opt/sdk")
	first := replaceInitializationBlock("export EDITOR=vim\n", block)
	second := replaceInitializationBlock(first, block)
	if first != second {
		t.Fatalf("initialization block changed on second write:\n%s", second)
	}
	if strings.Count(second, setupStart) != 1 {
		t.Fatalf("setup start count = %d", strings.Count(second, setupStart))
	}
}

func TestUpdateTargetsContainsManagedToolsOnly(t *testing.T) {
	home := t.TempDir()
	stateStore := store.New(home)
	state := model.State{
		Defaults: map[model.Tool]string{},
		Installed: map[model.Tool][]model.InstalledVersion{
			model.Java: {{Version: "21.0.12"}},
			model.Go:   {{Version: "1.26.5"}},
		},
	}
	if err := stateStore.Save(state); err != nil {
		t.Fatal(err)
	}
	targets, err := updateTargets(stateStore, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(targets, []model.Tool{model.Java, model.Go}) {
		t.Fatalf("targets = %v", targets)
	}
}

func TestOldManagedVersionsExcludesDefaultAndExternalInstalls(t *testing.T) {
	installed := []model.InstalledVersion{
		{Version: "21.0.12", Managed: true},
		{Version: "17.0.20", Managed: true},
		{Version: "11.0.32", Managed: false},
	}
	candidates := oldManagedVersions(installed, "25.0.4", "21.0.12")
	if !reflect.DeepEqual(candidates, []model.InstalledVersion{{Version: "17.0.20", Managed: true}}) {
		t.Fatalf("cleanup candidates = %v", candidates)
	}
}
