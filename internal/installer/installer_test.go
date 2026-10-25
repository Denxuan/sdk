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
	if err := tarWriter.WriteHeader(&tar.Header{Name: "tool-1.0/lib/npm.js", Mode: 0644, Size: 2}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write([]byte("ok")); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.WriteHeader(&tar.Header{Name: "tool-1.0/bin/npm", Typeflag: tar.TypeSymlink, Linkname: "../lib/npm.js"}); err != nil {
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
	contents, err := os.ReadFile(filepath.Join(root, "bin", "npm"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "ok" {
		t.Fatalf("contents = %q", contents)
	}
}

func TestSafeSymlinkTargetRejectsEscapingArchive(t *testing.T) {
	root := t.TempDir()
	if err := safeSymlinkTarget(root, filepath.Join(root, "bin", "npm"), "../../outside"); err == nil {
		t.Fatal("unsafe symbolic link was accepted")
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

func TestDownloadResumesAfterInterruptedRequest(t *testing.T) {
	const content = "0123456789"
	requests := 0
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if requests == 1 {
			return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Length": {"10"}}, ContentLength: 10, Body: io.NopCloser(&interruptedReader{content: content[:4]}), Request: request}, nil
		}
		if got := request.Header.Get("Range"); got != "bytes=4-" {
			t.Errorf("Range = %q, want bytes=4-", got)
		}
		return &http.Response{
			StatusCode:    http.StatusPartialContent,
			Header:        http.Header{"Content-Range": {"bytes 4-9/10"}, "Content-Length": {"6"}},
			ContentLength: 6,
			Body:          io.NopCloser(bytes.NewBufferString(content[4:])),
			Request:       request,
		}, nil
	})}

	installer := New()
	installer.HTTP = client
	installer.MaxAttempts = 1
	target := filepath.Join(t.TempDir(), "partial")
	if err := installer.download(context.Background(), "https://downloads.example.test/package", target); err == nil {
		t.Fatal("interrupted download unexpectedly succeeded")
	}
	partial, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(partial) != content[:4] {
		t.Fatalf("partial download = %q, want %q", partial, content[:4])
	}

	var reports []Progress
	installer.Progress = func(progress Progress) { reports = append(reports, progress) }
	if err := installer.download(context.Background(), "https://downloads.example.test/package", target); err != nil {
		t.Fatal(err)
	}
	complete, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(complete) != content {
		t.Fatalf("resumed download = %q, want %q", complete, content)
	}
	if len(reports) == 0 || reports[0] != (Progress{Downloaded: 4, Total: 10}) {
		t.Fatalf("initial resumed progress = %+v, want 4/10", reports)
	}
}

func TestCleanPartialDownloadsRemovesOnlyPartialFiles(t *testing.T) {
	root := t.TempDir()
	partialPaths := []string{
		filepath.Join(root, partialDownloadPrefix+"one"),
		filepath.Join(root, "nodejs", partialDownloadPrefix+"two"),
	}
	for _, path := range partialPaths {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("partial"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	keptPath := filepath.Join(root, "nodejs", "24.22.1", "bin", "node")
	if err := os.MkdirAll(filepath.Dir(keptPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keptPath, []byte("installed"), 0644); err != nil {
		t.Fatal(err)
	}

	removed, err := CleanPartialDownloads(root)
	if err != nil {
		t.Fatal(err)
	}
	if removed != len(partialPaths) {
		t.Fatalf("removed = %d, want %d", removed, len(partialPaths))
	}
	for _, path := range partialPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("partial file %s still exists", path)
		}
	}
	if _, err := os.Stat(keptPath); err != nil {
		t.Fatalf("installed file was removed: %v", err)
	}
}

type interruptedReader struct {
	content string
	done    bool
}

func (r *interruptedReader) Read(data []byte) (int, error) {
	if r.done {
		return 0, io.ErrUnexpectedEOF
	}
	r.done = true
	return copy(data, r.content), nil
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
