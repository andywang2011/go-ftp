package ftp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPlanDownloadResume_complete(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 100
	c := &ftpClient{conn: fc, transferBuf: 1024}
	plan, err := c.planDownloadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err != nil || !plan.done {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanDownloadResume_partial(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 100
	c := &ftpClient{conn: fc, transferBuf: 1024}
	plan, err := c.planDownloadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err != nil || plan.done || plan.offset != 40 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanDownloadResume_truncateWhenLocalLarger(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 100
	c := &ftpClient{conn: fc, transferBuf: 1024}
	plan, err := c.planDownloadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err != nil || !plan.truncateLocal || plan.offset != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanDownloadResume_sizeUnsupported(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizeErr["/remote"] = errors.New("SIZE not supported")
	c := &ftpClient{conn: fc, transferBuf: 1024}
	_, err := c.planDownloadResume(context.Background(), "/remote", "", applyTransferOpts([]TransferOption{WithResumeOffset(10)}, 1024))
	if !errors.Is(err, ErrResumeNotSupported) {
		t.Fatalf("err=%v", err)
	}
}

func TestPlanUploadResume_mismatch(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 50), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 80
	c := &ftpClient{conn: fc, transferBuf: 1024}
	_, err := c.planUploadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024), nil)
	if !errors.Is(err, ErrResumeOffsetMismatch) {
		t.Fatalf("err=%v", err)
	}
}

func TestPlanUploadResume_complete(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 50), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	fc.fileSizes["/remote"] = 50
	c := &ftpClient{conn: fc, transferBuf: 1024}
	plan, err := c.planUploadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024), nil)
	if err != nil || !plan.done {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanUploadResume_remoteMissing(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(local, make([]byte, 50), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := newFakeFTPConn()
	c := &ftpClient{conn: fc, transferBuf: 1024}
	plan, err := c.planUploadResume(context.Background(), "/remote", local, applyTransferOpts([]TransferOption{WithResume(true)}, 1024), nil)
	if err != nil || plan.done || plan.offset != 0 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestDownloadWriter_resumeNotSeekable(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 10
	c := &ftpClient{conn: fc, transferBuf: 1024}
	var buf bytesBuffer
	err := c.DownloadWriter(context.Background(), "/r", &buf, WithResumeOffset(5))
	if !errors.Is(err, ErrResumeNotSeekable) {
		t.Fatalf("err=%v", err)
	}
}

type bytesBuffer struct{ b []byte }

func (b *bytesBuffer) Write(p []byte) (int, error) {
	b.b = append(b.b, p...)
	return len(p), nil
}
