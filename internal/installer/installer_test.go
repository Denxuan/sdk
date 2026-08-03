package installer

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractTarGzipAndFindRoot(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "tool.tar.gz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gzipWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "tool-1.0/bin/tool", Mode: 0755, Size: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	extracted := filepath.Join(dir, "extracted")
	if err := extract(archive, extracted); err != nil {
		t.Fatal(err)
	}
	root, err := archiveRoot(extracted)
	if err != nil {
		t.Fatal(err)
	}
	if root != filepath.Join(extracted, "tool-1.0") {
		t.Fatalf("root = %s", root)
	}
	contents, err := os.ReadFile(filepath.Join(root, "bin", "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "ok" {
		t.Fatalf("contents = %q", contents)
	}
}

func TestSafePathRejectsTraversal(t *testing.T) {
	if _, err := safePath(t.TempDir(), "../../outside"); err == nil {
		t.Fatal("unsafe path was accepted")
	}
}

func TestProgressWriterReportsDownloadSize(t *testing.T) {
	var reports []Progress
	writer := &progressWriter{writer: io.Discard, total: 5, report: func(progress Progress) { reports = append(reports, progress) }}
	if _, err := writer.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Fatalf("report count = %d", len(reports))
	}
	if reports[0] != (Progress{Downloaded: 5, Total: 5}) {
		t.Fatalf("progress = %+v", reports[0])
	}
}

func TestDownloadRetriesTemporaryServerFailure(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Status: "503 Service Unavailable", Body: io.NopCloser(bytes.NewBufferString("temporary failure"))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewBufferString("package"))}, nil
	})}

	installer := New()
	installer.HTTP = client
	installer.MaxAttempts = 2
	installer.RetryDelay = func(int) time.Duration { return 0 }
	retries := 0
	installer.Retry = func(attempt, total int, err error) { retries++ }
	target := filepath.Join(t.TempDir(), "package")
	if err := installer.download(context.Background(), "https://downloads.example.test/package", target); err != nil {
		t.Fatal(err)
	}
	if requests != 2 || retries != 1 {
		t.Fatalf("requests = %d, retries = %d", requests, retries)
	}
	contents, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "package" {
		t.Fatalf("contents = %q", contents)
	}
}

func TestVerifyChecksumRejectsModifiedArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "package")
	if err := os.WriteFile(path, []byte("original package"), 0644); err != nil {
		t.Fatal(err)
	}
	expectedHash := sha256.Sum256([]byte("original package"))
	checksum := Checksum{Algorithm: "sha256", Value: hex.EncodeToString(expectedHash[:])}
	if err := verifyChecksum(path, checksum); err != nil {
		t.Fatalf("valid checksum failed: %v", err)
	}
	if err := os.WriteFile(path, []byte("modified package"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(path, checksum); err == nil {
		t.Fatal("modified archive passed checksum verification")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}
