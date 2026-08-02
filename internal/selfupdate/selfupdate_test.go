package selfupdate

import "testing"

func TestSelectAssetUsesCurrentPlatformName(t *testing.T) {
	assets := []Asset{
		{Name: "sdk_0.0.2_linux_amd64.tar.gz"},
		{Name: "sdk_0.0.2_darwin_arm64.tar.gz"},
		{Name: "checksums.txt"},
	}
	asset, err := selectAsset(assets, "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if asset.Name != "sdk_0.0.2_darwin_arm64.tar.gz" {
		t.Fatalf("asset = %s", asset.Name)
	}
}

func TestNormalizeTag(t *testing.T) {
	if got := normalizeTag("0.0.2"); got != "v0.0.2" {
		t.Fatalf("tag = %s", got)
	}
	if got := normalizeTag("v0.0.2"); got != "v0.0.2" {
		t.Fatalf("tag = %s", got)
	}
}
