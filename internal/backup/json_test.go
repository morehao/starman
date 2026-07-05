package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadWriteDB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	data := []byte("fake sqlite data")

	if err := WriteDB(path, data); err != nil {
		t.Fatal(err)
	}

	got, err := ReadDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %q, got %q", data, got)
	}
}

func TestReadDB_NotFound(t *testing.T) {
	_, err := ReadDB("/nonexistent/path/test.db")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteDB_PermissionError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nowrite", "test.db")
	err := WriteDB(path, []byte("test"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestReadWriteDB_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.db")

	if err := WriteDB(path, []byte{}); err != nil {
		t.Fatal(err)
	}
	got, err := ReadDB(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d bytes", len(got))
	}
}

func TestReadWriteDB_PreservePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "perm.db")
	data := []byte("test")

	if err := WriteDB(path, data); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}
