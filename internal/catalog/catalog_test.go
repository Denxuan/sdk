package catalog

import (
	"reflect"
	"strings"
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

func TestUniqueExcludesPrereleaseVersions(t *testing.T) {
	got := unique([]string{"4.0.0", "4.0.0-rc-1", "4.0.0-M1", "4.0.0-beta-1", "3.9.16"})
	want := []string{"4.0.0", "3.9.16"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unique() = %v, want %v", got, want)
	}
}

func TestJavaAssetsURLRequestsOneLatestRelease(t *testing.T) {
	got := javaAssetsURL(26, "mac", "aarch64")
	for _, expected := range []string{"feature_releases/26/ga", "architecture=aarch64", "page_size=1", "sort_order=DESC"} {
		if !strings.Contains(got, expected) {
			t.Errorf("URL does not contain %q: %s", expected, got)
		}
	}
}

func TestStatusErrorIncludesServerStatus(t *testing.T) {
	err := (&StatusError{URL: "https://example.test", StatusCode: 404, Status: "404 Not Found"}).Error()
	if !strings.Contains(err, "404 Not Found") {
		t.Fatalf("error = %q", err)
	}
}

func TestManifestChecksumFindsRequestedNodeArchive(t *testing.T) {
	manifest := "abc123  node-v24.0.0-darwin-arm64.tar.gz\n" +
		"def456  node-v24.0.0-linux-x64.tar.gz\n"
	checksum, found := manifestChecksum(manifest, "node-v24.0.0-darwin-arm64.tar.gz")
	if !found || checksum != "abc123" {
		t.Fatalf("manifest checksum = %q, found = %t", checksum, found)
	}
}
