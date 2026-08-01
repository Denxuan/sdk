package catalog

import (
	"reflect"
	"testing"
)

func TestUniqueSortsReleaseVersionsNumerically(t *testing.T) {
	got := unique([]string{"3.9.2", "3.9.16", "26", "8", "21", "3.9.16"})
	want := []string{"26", "21", "8", "3.9.16", "3.9.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unique() = %v, want %v", got, want)
	}
}

func TestStableVersionPrecedesPrerelease(t *testing.T) {
	if versionCompare("4.0.0", "4.0.0-rc-1") <= 0 {
		t.Fatal("stable release did not sort before its release candidate")
	}
}
