package ftp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewClient_connectHook(t *testing.T) {
	old := connectFTP
	defer func() { connectFTP = old }()

	connectFTP = func(Config) (ftpConn, error) {
		return nil, fmt.Errorf("ftp dial: %w", errors.New("dial fail"))
	}
	_, err := NewClient(Config{Host: "h", User: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected dial error")
	}

	connectFTP = func(Config) (ftpConn, error) {
		return nil, fmt.Errorf("ftp login: %w", errors.New("login fail"))
	}
	_, err = NewClient(Config{Host: "h", User: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected login error")
	}

	fc := newFakeFTPConn()
	connectFTP = func(cfg Config) (ftpConn, error) {
		return fc, nil
	}
	cli, err := NewClient(Config{Host: "h", Port: 21, User: "u", Password: "p", TransferBufferSize: 4096})
	if err != nil {
		t.Fatal(err)
	}
	c := cli.(*ftpClient)
	if c.transferBuf != 4096 {
		t.Fatal(c.transferBuf)
	}
	if err := cli.Close(); err != nil {
		t.Fatal(err)
	}
	if c.conn != nil {
		t.Fatal("conn should be nil after close")
	}
}

func TestClose_quitError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.quitErr = errors.New("quit fail")
	c := testClient(fc, 1024)
	if err := c.Close(); err == nil {
		t.Fatal("expected quit error")
	}
}

func TestConfig_transferBufferSize(t *testing.T) {
	if n := (Config{TransferBufferSize: 8192}).transferBufferSize(); n != 8192 {
		t.Fatal(n)
	}
	if n := (Config{}).transferBufferSize(); n != defaultTransferBuffer {
		t.Fatal(n)
	}
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

func TestDownload_errors(t *testing.T) {
	var nilClient *ftpClient
	if err := nilClient.Download(context.Background(), "/r", "/l"); err == nil {
		t.Fatal()
	}
	c := testClient(newFakeFTPConn(), 1024)
	c.conn = nil
	if err := c.Download(context.Background(), "/r", filepath.Join(t.TempDir(), "x")); err == nil {
		t.Fatal()
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

type brokenSeeker struct{ io.Reader }

func (brokenSeeker) Write(p []byte) (int, error) { return len(p), nil }
func (brokenSeeker) Seek(int64, int) (int64, error) {
	return 0, errors.New("seek fail")
}

func TestUpload_errors(t *testing.T) {
	var nilClient *ftpClient
	if err := nilClient.Upload(context.Background(), "/l", "/r"); err == nil {
		t.Fatal()
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

func TestRemoteUploadedSize_otherError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.fileSizeErr["/r"] = errors.New("size fail")
	c := testClient(fc, 1024)
	_, err := c.remoteUploadedSize("/r")
	if !errors.Is(err, ErrResumeNotSupported) {
		t.Fatal(err)
	}
}

type readOnly struct{ io.Reader }

func TestSeekReaderTo_notSeekable(t *testing.T) {
	if err := seekReaderTo(readOnly{bytes.NewReader(nil)}, 0); !errors.Is(err, ErrResumeNotSeekable) {
		t.Fatal(err)
	}
}

func TestCopyWithContext_shortWrite(t *testing.T) {
	_, err := copyWithContext(context.Background(), shortWriter{}, strings.NewReader("ab"), make([]byte, 8))
	if err == nil {
		t.Fatal()
	}
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return 0, nil }

func TestJlaffayeConn_RetrFrom_mockServer(t *testing.T) {
	mock, port := startFTPMock(t)
	defer mock.Close()

	cfg := Config{
		Host:    "127.0.0.1",
		Port:    port,
		User:    "u",
		Password: "p",
		Timeout: 5 * time.Second,
	}
	old := connectFTP
	connectFTP = defaultConnectFTP
	defer func() { connectFTP = old }()

	cli, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()

	dir := t.TempDir()
	local := filepath.Join(dir, "out")
	if err := cli.Download(context.Background(), "/magic-file", local); err != nil {
		t.Fatal(err)
	}
}
