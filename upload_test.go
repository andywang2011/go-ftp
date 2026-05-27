package ftp

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUpload_resume(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "in.bin")
	data := make([]byte, 100)
	for i := range data {
		data[i] = byte(i)
	}
	if err := os.WriteFile(local, data, 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 40
	fc.storData = make([]byte, 40)
	copy(fc.storData, data[:40])

	c := &ftpClient{conn: fc, transferBuf: 32}
	if err := c.Upload(context.Background(), local, "/remote", WithResume(true)); err != nil {
		t.Fatal(err)
	}
	if len(fc.storOffsets) == 0 || fc.storOffsets[0] != 40 {
		t.Fatalf("stor offsets=%v", fc.storOffsets)
	}
	if len(fc.storData) != 100 {
		t.Fatalf("remote len=%d", len(fc.storData))
	}
	if !bytes.Equal(fc.storData, data) {
		t.Fatal("content mismatch")
	}
}

func TestUpload_resume_alreadyComplete(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "in.bin")
	if err := os.WriteFile(local, make([]byte, 50), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 50
	c := &ftpClient{conn: fc, transferBuf: 32}
	if err := c.Upload(context.Background(), local, "/remote", WithResume(true)); err != nil {
		t.Fatal(err)
	}
	if len(fc.storOffsets) != 0 {
		t.Fatalf("expected no stor, got %v", fc.storOffsets)
	}
}

func TestUploadReader_resumeSeeker(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 10
	fc.storData = make([]byte, 10)
	payload := bytes.Repeat([]byte("a"), 30)
	r := bytes.NewReader(payload)
	c := &ftpClient{conn: fc, transferBuf: 16}
	if err := c.UploadReader(context.Background(), r, "/remote", WithResume(true)); err != nil {
		t.Fatal(err)
	}
	if fc.storOffsets[0] != 10 {
		t.Fatalf("offset=%v", fc.storOffsets)
	}
	if len(fc.storData) != 30 {
		t.Fatalf("len=%d", len(fc.storData))
	}
}

func TestUpload_errors(t *testing.T) {
	var nilClient *ftpClient
	if err := nilClient.Upload(context.Background(), "/l", "/r"); !errors.Is(err, ErrNotConnected) {
		t.Fatal(err)
	}
	c := testClient(newFakeFTPConn(), 1024)
	if err := c.Upload(context.Background(), "/no-such", "/r"); err == nil {
		t.Fatal()
	}
}

func TestUpload_storError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.storErr = errors.New("stor fail")
	dir := t.TempDir()
	local := filepath.Join(dir, "in")
	_ = os.WriteFile(local, []byte("data"), 0o644)
	c := testClient(fc, 1024)
	if err := c.Upload(context.Background(), local, "/r"); err == nil {
		t.Fatal()
	}
	fc.storErr = errors.New("stor fail")
	if err := c.Upload(context.Background(), local, "/r", WithResume(true)); err == nil {
		t.Fatal()
	}
}

func TestUploadReader_seekFail(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 1
	c := testClient(fc, 1024)
	err := c.UploadReader(context.Background(), brokenSeeker{bytes.NewReader(nil)}, "/r", WithResume(true))
	if err == nil {
		t.Fatal(err)
	}
}

func TestUploadAt_storPlainError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.storErr = errors.New("stor")
	c := testClient(fc, 1024)
	err := c.uploadAt(context.Background(), bytes.NewReader([]byte("x")), "/r", "", WithBufferSize(1024))
	if err == nil {
		t.Fatal()
	}
}

func TestUploadAt_nilConn(t *testing.T) {
	var c *ftpClient
	if err := c.uploadAt(context.Background(), nil, "/r", "", WithBufferSize(1024)); !errors.Is(err, ErrNotConnected) {
		t.Fatal(err)
	}
}

func TestUploadAt_resumeEnabledZeroOffset(t *testing.T) {
	fc := newFakeFTPConn()
	dir := t.TempDir()
	local := filepath.Join(dir, "in")
	_ = os.WriteFile(local, []byte("hello"), 0o644)
	c := testClient(fc, 1024)
	if err := c.Upload(context.Background(), local, "/r", WithResume(true)); err != nil {
		t.Fatal(err)
	}
}
