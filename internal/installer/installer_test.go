package installer

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
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
