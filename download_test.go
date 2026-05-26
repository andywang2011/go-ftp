package ftp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDownload_resume(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(local, make([]byte, 30), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 100
	c := &ftpClient{conn: fc, transferBuf: 32}
	if err := c.Download(context.Background(), "/remote", local, WithResume(true)); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(local)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() != 100 {
		t.Fatalf("size=%d", st.Size())
	}
	if len(fc.retrOffsets) == 0 || fc.retrOffsets[0] != 30 {
		t.Fatalf("retr offsets=%v", fc.retrOffsets)
	}
}

func TestDownload_resume_alreadyComplete(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "out.bin")
	if err := os.WriteFile(local, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 100
	c := &ftpClient{conn: fc, transferBuf: 32}
	if err := c.Download(context.Background(), "/remote", local, WithResume(true)); err != nil {
		t.Fatal(err)
	}
	if len(fc.retrOffsets) != 0 {
		t.Fatalf("expected no retr, got %v", fc.retrOffsets)
	}
}

func TestDownloadWriter_resumeOffset(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 20
	c := &ftpClient{conn: fc, transferBuf: 8}
	f, err := os.Create(filepath.Join(t.TempDir(), "w.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = f.Write(make([]byte, 5)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err = c.DownloadWriter(context.Background(), "/remote", f, WithResumeOffset(5)); err != nil {
		t.Fatal(err)
	}
	st, _ := f.Stat()
	if st.Size() != 20 {
		t.Fatalf("size=%d", st.Size())
	}
	if fc.retrOffsets[0] != 5 {
		t.Fatalf("offset=%v", fc.retrOffsets)
	}
}
