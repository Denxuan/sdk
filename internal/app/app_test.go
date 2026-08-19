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
		err := Run(context.Background(), args, &out)
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

func TestNormalizeJavaVersionDefaultsToTemurin(t *testing.T) {
	if got := normalizeToolVersion(model.Java, "26.0.2"); got != "26.0.2-tem" {
		t.Fatalf("normalized Java version = %q", got)
	}
	if got := normalizeToolVersion(model.Java, "26.0.2-zulu"); got != "26.0.2-zulu" {
		t.Fatalf("Zulu Java version changed = %q", got)
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

func TestAskToMigrateConfiguration(t *testing.T) {
	var out bytes.Buffer
	migrate, err := askToMigrateConfiguration(&out, strings.NewReader("y\n"), model.NodeJS, "20.0.0", "22.0.0")
	if err != nil || !migrate {
		t.Fatalf("migration answer = %t, %v", migrate, err)
	}
	if !strings.Contains(out.String(), "Migrate nodejs configuration from 20.0.0 to 22.0.0?") {
		t.Fatalf("prompt = %q", out.String())
	}
}

func TestReplaceInitializationBlockIsIdempotent(t *testing.T) {
	block := zshInitialization("/opt/sdk")
	if !strings.Contains(block, "export SDK_HOME=\"/opt/sdk\"") || !strings.Contains(block, "source \"$SDK_HOME/init.zsh\"") {
		t.Fatalf("initialization does not source the SDK shell script:\n%s", block)
	}
	script := zshScript("/opt/sdk/bin/sdk")
	if !strings.Contains(script, "sdk()") || !strings.Contains(script, "default|use|install|update") || !strings.Contains(script, "add-zsh-hook chpwd") {
		t.Fatalf("shell script does not define the sdk shell function:\n%s", script)
	}
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

func TestDescribeUpdateCheck(t *testing.T) {
	tests := []struct {
		name      string
		installed []model.InstalledVersion
		current   string
		latest    string
		want      string
	}{
		{name: "update available", current: "1.25.0", latest: "1.26.5", want: "go: update available 1.25.0 -> 1.26.5"},
		{name: "already installed", installed: []model.InstalledVersion{{Version: "1.26.5"}}, current: "1.25.0", latest: "1.26.5", want: "go: update to 1.26.5 is already installed; run sdk default go 1.26.5 to use it"},
		{name: "up to date", current: "1.26.5", latest: "1.26.5", want: "go: up to date (1.26.5)"},
		{name: "current newer", current: "26.0.2", latest: "25.0.4", want: "go: current version 26.0.2 is newer than the recommended version 25.0.4"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := describeUpdateCheck(model.Go, test.installed, test.current, test.latest); got != test.want {
				t.Fatalf("description = %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseUpdateArgs(t *testing.T) {
	for _, args := range [][]string{{"--check"}, {"go", "--check"}, {"--check", "go"}} {
		tools, checkOnly, err := parseUpdateArgs(args)
		if err != nil || !checkOnly || len(tools) > 1 {
			t.Fatalf("parseUpdateArgs(%v) = %v, %t, %v", args, tools, checkOnly, err)
		}
	}
}
