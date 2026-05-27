package ftp

import (
	"bytes"
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

func TestWrapResumeNotSupported_nil(t *testing.T) {
	if !errors.Is(wrapResumeNotSupported(nil), ErrResumeNotSupported) {
		t.Fatal()
	}
}

func TestIsFTPFileUnavailable_false(t *testing.T) {
	if isFTPFileUnavailable(errors.New("other")) {
		t.Fatal()
	}
}

func TestPlanDownloadResume_ctxAndNoLocal(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.planDownloadResume(ctx, "/r", "", applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err == nil {
		t.Fatal()
	}
	_, err = c.planDownloadResume(context.Background(), "/r", "", applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPlanDownloadResumeStream_doneAndTruncate(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 5
	c := testClient(fc, 1024)
	f, _ := os.Create(filepath.Join(t.TempDir(), "w"))
	defer f.Close()
	plan, err := c.planDownloadResumeStream(context.Background(), "/r",
		applyTransferOpts([]TransferOption{WithResumeOffset(5)}, 1024), f)
	if err != nil || !plan.done {
		t.Fatalf("expected done plan=%+v err=%v", plan, err)
	}
	_, _ = f.Seek(0, 0)
	plan, err = c.planDownloadResumeStream(context.Background(), "/r",
		applyTransferOpts([]TransferOption{WithResumeOffset(10)}, 1024), f)
	if err != nil || plan.offset != 0 {
		t.Fatalf("truncate plan=%+v err=%v", plan, err)
	}
}

func TestPlanDownloadResumeStream_ctxCancel(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.planDownloadResumeStream(ctx, "/r",
		applyTransferOpts([]TransferOption{WithResumeOffset(1)}, 1024), &bytesSink{})
	if err == nil {
		t.Fatal()
	}
}

func TestPlanDownloadResume_explicitNoLocal(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 20
	c := testClient(fc, 1024)
	plan, err := c.planDownloadResume(context.Background(), "/r", "",
		applyTransferOpts([]TransferOption{WithResumeOffset(3)}, 1024))
	if err != nil || plan.offset != 3 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
}

func TestPlanUploadResume_ctx(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.planUploadResume(ctx, "/r", "", applyTransferOpts(nil, 1024), nil)
	if err == nil {
		t.Fatal()
	}
}

func TestLocalFileSize_statError(t *testing.T) {
	_, err := localFileSize("/no/such/file", nil, transferOpts{resumeAuto: true})
	if err == nil {
		t.Fatal()
	}
}

func TestSeekReaderTo_success(t *testing.T) {
	r := bytes.NewReader([]byte("abc"))
	if err := seekReaderTo(r, 1); err != nil {
		t.Fatal(err)
	}
}

func TestSeekReaderTo_notSeekable(t *testing.T) {
	if err := seekReaderTo(readOnly{bytes.NewReader(nil)}, 0); !errors.Is(err, ErrResumeNotSeekable) {
		t.Fatal(err)
	}
}

func TestVerifyDownloadSize_statFail(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	err := c.verifyDownloadSize(context.Background(), "/r", "/no/file", resumePlan{remoteSize: 1})
	if err == nil {
		t.Fatal()
	}
}

func TestVerifyDownloadSize_paths(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 10
	c := testClient(fc, 1024)
	dir := t.TempDir()
	local := filepath.Join(dir, "f")
	_ = os.WriteFile(local, make([]byte, 5), 0o644)
	err := c.verifyDownloadSize(context.Background(), "/r", local, resumePlan{remoteSize: 10})
	if err == nil {
		t.Fatal()
	}
	fc.fileSizes["/r"] = 5
	err = c.verifyDownloadSize(context.Background(), "/r", local, resumePlan{remoteSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.verifyDownloadSize(context.Background(), "/r", "", resumePlan{}); err != nil {
		t.Fatal(err)
	}
}

func TestVerifyDownloadSize_remoteSizeZero(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 8
	c := testClient(fc, 1024)
	dir := t.TempDir()
	local := filepath.Join(dir, "f")
	_ = os.WriteFile(local, make([]byte, 8), 0o644)
	if err := c.verifyDownloadSize(context.Background(), "/r", local, resumePlan{}); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteUploadedSize_otherError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizeErr["/r"] = errors.New("size fail")
	c := testClient(fc, 1024)
	_, err := c.remoteUploadedSize("/r")
	if !errors.Is(err, ErrResumeNotSupported) {
		t.Fatal(err)
	}
}

func TestLocalFileSize_noResume(t *testing.T) {
	n, err := localFileSize("", nil, transferOpts{})
	if err != nil || n != 0 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestPlanDownloadResume_statError(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	dir := filepath.Join(t.TempDir(), "missing-parent", "sub", "f")
	_, err := c.planDownloadResume(context.Background(), "/r", dir, applyTransferOpts([]TransferOption{WithResume(true)}, 1024))
	if err == nil {
		t.Fatal()
	}
}

func TestPlanDownloadResumeStream_seekFail(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizes["/r"] = 10
	c := testClient(fc, 1024)
	_, err := c.planDownloadResumeStream(context.Background(), "/r", applyTransferOpts([]TransferOption{WithResumeOffset(1)}, 1024), brokenSeeker{})
	if err == nil {
		t.Fatal()
	}
}
