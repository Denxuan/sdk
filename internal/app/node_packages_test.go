package app

import (
	"reflect"
	"testing"
)

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
