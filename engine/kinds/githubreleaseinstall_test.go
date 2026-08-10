package kinds

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for name, content := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}

func writeTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	defer zw.Close()
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExtractFromTarGzFindsBinaryByBaseName(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	writeTestTarGz(t, archivePath, map[string]string{
		"README.md": "not the binary",
		"lazygit":   "fake binary content",
	})

	dest := filepath.Join(dir, "out")
	if err := extractFromTarGz(archivePath, "lazygit", dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake binary content" {
		t.Fatalf("got %q", data)
	}
}

func TestExtractFromTarGzErrorsWhenBinaryMissing(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.tar.gz")
	writeTestTarGz(t, archivePath, map[string]string{"README.md": "nothing else here"})

	if err := extractFromTarGz(archivePath, "lazygit", filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected an error when the wanted binary isn't in the archive")
	}
}

func TestExtractFromZipFindsBinaryByBaseName(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.zip")
	writeTestZip(t, archivePath, map[string]string{"procs": "fake procs binary"})

	dest := filepath.Join(dir, "out")
	if err := extractFromZip(archivePath, "procs", dest); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "fake procs binary" {
		t.Fatalf("got %q", data)
	}
}

func TestExtractFromZipErrorsWhenBinaryMissing(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "test.zip")
	writeTestZip(t, archivePath, map[string]string{"other-file": "irrelevant"})

	if err := extractFromZip(archivePath, "procs", filepath.Join(dir, "out")); err == nil {
		t.Fatal("expected an error when the wanted binary isn't in the archive")
	}
}
