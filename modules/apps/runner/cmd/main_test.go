package main

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractEmployeeID_ValidArchive(t *testing.T) {
	// Create a temporary valid tar.gz archive with metadata.json
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test-emp.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create test archive: %v", err)
	}

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	metaContent := []byte(`{"schema_version":"1.0","session_id":"s-1","employee_id":"emp-test-42"}`)
	hdr := &tar.Header{
		Name: "metadata.json",
		Mode: 0600,
		Size: int64(len(metaContent)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("write tar header: %v", err)
	}
	if _, err := tw.Write(metaContent); err != nil {
		t.Fatalf("write tar content: %v", err)
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close file: %v", err)
	}

	empID, err := extractEmployeeID(archivePath)
	if err != nil {
		t.Fatalf("extractEmployeeID failed: %v", err)
	}
	if empID != "emp-test-42" {
		t.Errorf("got employeeID %q, want %q", empID, "emp-test-42")
	}
}

func TestExtractEmployeeID_MissingMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "empty.tar.gz")

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)
	_ = tw.Close()
	_ = gzw.Close()
	_ = f.Close()

	_, err = extractEmployeeID(archivePath)
	if err == nil {
		t.Fatal("expected error for missing metadata.json, got nil")
	}
}

func TestGetFreePort(t *testing.T) {
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("getFreePort failed: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Fatalf("invalid port returned: %d", port)
	}
}

func TestLoadRunnerConfig_Defaults(t *testing.T) {
	cfg, err := loadRunnerConfig()
	if err != nil {
		t.Fatalf("loadRunnerConfig failed: %v", err)
	}
	if cfg.serverURL == "" {
		t.Error("expected non-empty serverURL")
	}
	if cfg.concurrency <= 0 {
		t.Errorf("expected positive concurrency, got %d", cfg.concurrency)
	}
}
