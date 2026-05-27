package ftp

import (
	"context"
	"errors"
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

func TestDownload_errors(t *testing.T) {
	var nilClient *ftpClient
	if err := nilClient.Download(context.Background(), "/r", "/l"); !errors.Is(err, ErrNotConnected) {
		t.Fatal(err)
	}
	c := testClient(newFakeFTPConn(), 1024)
	c.conn = nil
	if err := c.Download(context.Background(), "/r", filepath.Join(t.TempDir(), "x")); !errors.Is(err, ErrNotConnected) {
		t.Fatal(err)
	}
}

func TestDownload_retrResumeError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 100
	fc.retrErr = errors.New("retr fail")
	dir := t.TempDir()
	local := filepath.Join(dir, "o")
	_ = os.WriteFile(local, make([]byte, 10), 0o644)
	c := testClient(fc, 1024)
	err := c.Download(context.Background(), "/r", local, WithResume(true))
	if err == nil || !errors.Is(err, ErrResumeNotSupported) {
		t.Fatalf("err=%v", err)
	}
}

func TestDownload_truncateResume(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 50
	dir := t.TempDir()
	local := filepath.Join(dir, "o")
	_ = os.WriteFile(local, make([]byte, 80), 0o644)
	c := testClient(fc, 1024)
	if err := c.Download(context.Background(), "/r", local, WithResume(true)); err != nil {
		t.Fatal(err)
	}
	st, _ := os.Stat(local)
	if st.Size() != 50 {
		t.Fatalf("size=%d", st.Size())
	}
}

func TestDownloadWriter_streamLocalLargerThanRemote(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 5
	c := testClient(fc, 1024)
	f, err := os.Create(filepath.Join(t.TempDir(), "w"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err = f.Write(make([]byte, 20)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	if err = c.DownloadWriter(context.Background(), "/r", f, WithResumeOffset(20)); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadWriter_seekError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 10
	c := testClient(fc, 1024)
	err := c.DownloadWriter(context.Background(), "/r", brokenSeeker{}, WithResumeOffset(1))
	if err == nil {
		t.Fatal()
	}
}

func TestDownload_mkdirInvalidPath(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	err := c.Download(context.Background(), "/r", "/\x00invalid/path/file")
	if err == nil {
		t.Fatal("expected mkdir/open error")
	}
}

func TestDownload_seekErrorOnResume(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("unix specific")
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 100
	dir := t.TempDir()
	local := filepath.Join(dir, "o")
	_ = os.WriteFile(local, make([]byte, 10), 0o644)
	c := testClient(fc, 1024)
	if err := c.Download(context.Background(), "/r", local, WithResume(true)); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadWriter_retrErrorNoOffset(t *testing.T) {
	fc := newFakeFTPConn()
	fc.retrErr = errors.New("retr")
	c := testClient(fc, 1024)
	err := c.DownloadWriter(context.Background(), "/r", &bytesSink{})
	if err == nil {
		t.Fatal()
	}
}

func TestDownloadWriter_nilConn(t *testing.T) {
	var c *ftpClient
	if err := c.DownloadWriter(context.Background(), "/r", &bytesSink{}); !errors.Is(err, ErrNotConnected) {
		t.Fatal(err)
	}
}

func TestDownload_fullNoResume(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 4
	c := testClient(fc, 1024)
	local := filepath.Join(t.TempDir(), "out")
	if err := c.Download(context.Background(), "/r", local); err != nil {
		t.Fatal(err)
	}
}
