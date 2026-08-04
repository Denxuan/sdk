package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Denxuan/sdk/internal/model"
)

func TestMigrateMavenConfigurationPreservesNewDefault(t *testing.T) {
	source := t.TempDir()
	destination := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(destination, "conf"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "conf", "settings.xml"), []byte("old settings"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "conf", "settings.xml"), []byte("new default"), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	migrateToolConfiguration(context.Background(), model.Maven, source, destination, &out)
	data, err := os.ReadFile(filepath.Join(destination, "conf", "settings.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old settings" {
		t.Fatalf("migrated settings = %q", data)
	}
	backup, err := os.ReadFile(filepath.Join(destination, "conf", "settings.xml.sdk-default"))
	if err != nil || string(backup) != "new default" {
		t.Fatalf("default backup = %q, %v", backup, err)
	}
	if !strings.Contains(out.String(), "Migrated Maven configuration conf/settings.xml") {
		t.Fatalf("migration output = %q", out.String())
	}
}

func TestParseNPMGlobalPackagesSkipsBundledPackages(t *testing.T) {
	packages, err := parseNPMGlobalPackages([]byte(`{
  "dependencies": {
    "npm": {"version": "11.16.0"},
    "corepack": {"version": "0.34.6"},
    "@scope/tool": {"version": "2.3.4"},
    "eslint": {"version": "9.0.0"}
  }
}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []npmGlobalPackage{{Name: "@scope/tool", Version: "2.3.4"}, {Name: "eslint", Version: "9.0.0"}}
	if !reflect.DeepEqual(packages, want) {
		t.Fatalf("packages = %#v, want %#v", packages, want)
	}
	if got := npmPackageSpec(packages[0]); got != "@scope/tool@2.3.4" {
		t.Fatalf("package spec = %q", got)
	}
}
