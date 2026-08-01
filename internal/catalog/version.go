package catalog

import (
	"sort"
	"strconv"
	"strings"
)

func unique(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	versions := make([]string, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versionCompare(versions[i], versions[j]) > 0 })
	return versions
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
