package catalog

import (
	"sort"
	"strconv"
	"strings"
)

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !isStableVersion(value) {
			continue
		}
		seen[value] = struct{}{}
	}
	versions := make([]string, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versionCompare(versions[i], versions[j]) > 0 })
	return versions
}

func stableReleases(values []Version) []Version {
	byNumber := make(map[string]Version, len(values))
	for _, version := range values {
		if !isStableVersion(version.Number) {
			continue
		}
		_, found := byNumber[version.Number]
		if !found || version.LTS {
			byNumber[version.Number] = version
		}
	}
	releases := make([]Version, 0, len(byNumber))
	for _, release := range byNumber {
		releases = append(releases, release)
	}
	sort.Slice(releases, func(i, j int) bool {
		return versionCompare(releases[i].Number, releases[j].Number) > 0
	})
	return releases
}

func stableVersions(values []string) []Version {
	releases := make([]Version, 0, len(values))
	for _, value := range values {
		releases = append(releases, Version{Number: value})
	}
	return stableReleases(releases)
}

// isStableVersion excludes labels commonly used by the four upstream
// catalogues for preview, milestone and release-candidate builds.
func isStableVersion(version string) bool {
	lowercase := strings.ToLower(version)
	for _, marker := range []string{"alpha", "beta", "rc", "snapshot", "preview", "nightly", "canary", "-ea"} {
		if strings.Contains(lowercase, marker) {
			return false
		}
	}
	for _, part := range versionParts(version) {
		if len(part) >= 2 && (part[0] == 'M' || part[0] == 'm') {
			if _, err := strconv.Atoi(part[1:]); err == nil {
				return false
			}
		}
	}
	return true
}

func versionCompare(left, right string) int {
	leftParts := versionParts(left)
	rightParts := versionParts(right)
	for index := 0; index < min(len(leftParts), len(rightParts)); index++ {
		if result := comparePart(leftParts[index], rightParts[index]); result != 0 {
			return result
		}
	}
	if len(leftParts) == len(rightParts) {
		return 0
	}
	if strings.Contains(left, "-") != strings.Contains(right, "-") {
		if strings.Contains(left, "-") {
			return -1
		}
		return 1
	}
	if len(leftParts) > len(rightParts) {
		return 1
	}
	return -1
}

func versionParts(version string) []string {
	return strings.FieldsFunc(strings.TrimPrefix(version, "v"), func(character rune) bool {
		return character == '.' || character == '-' || character == '+' || character == '_'
	})
}

func comparePart(left, right string) int {
	leftNumber, leftIsNumber := strconv.Atoi(left)
	rightNumber, rightIsNumber := strconv.Atoi(right)
	if leftIsNumber == nil && rightIsNumber == nil {
		if leftNumber > rightNumber {
			return 1
		}
		if leftNumber < rightNumber {
			return -1
		}
		return 0
	}
	if left > right {
		return 1
	}
	if left < right {
		return -1
	}
	return 0
}
