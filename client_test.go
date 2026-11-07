package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
)

type fakeClient struct {
	uploadErr   error
	downloadErr error
	listErr     error
	closeErr    error
	listResult  []string
}

func (f *fakeClient) Upload(ctx context.Context, localPath, remotePath string, _ ...TransferOption) error {
	if f.uploadErr != nil {
		return f.uploadErr
	}
	if localPath == "" || remotePath == "" {
		return errors.New("ftp: empty path")
	}
	return ctx.Err()
}

func (f *fakeClient) UploadReader(ctx context.Context, _ io.Reader, remotePath string, _ ...TransferOption) error {
	if remotePath == "" {
		return errors.New("ftp: empty path")
	}
	return ctx.Err()
}

func (f *fakeClient) Download(ctx context.Context, _, _ string, _ ...TransferOption) error {
	return f.downloadErr
}

func (f *fakeClient) DownloadWriter(ctx context.Context, _ string, _ io.Writer, _ ...TransferOption) error {
	return f.downloadErr
}

func (f *fakeClient) List(ctx context.Context, _ string) ([]string, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listResult, nil
}

func (f *fakeClient) ListEach(ctx context.Context, _ string, fn func(Entry) error) error {
	if f.listErr != nil {
		return f.listErr
	}
	for _, name := range f.listResult {
		if err := fn(Entry{Name: name}); err != nil {
			if errors.Is(err, ErrStopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

func (f *fakeClient) ListIter(context.Context, string) (*ListIter, error) {
	return nil, f.listErr
}

func (f *fakeClient) WalkEach(context.Context, string, func(Entry) error) error {
	return f.listErr
}

func (f *fakeClient) Close() error {
	return f.closeErr
}

func TestFakeClient_implementsClient(t *testing.T) {
	var _ Client = (*fakeClient)(nil)
}

func TestFakeClient_Upload(t *testing.T) {
	c := &fakeClient{}
	if err := c.Upload(context.Background(), "/local", "/remote"); err != nil {
		t.Fatal(err)
	}
}

func TestFTPClient_Close_nil(t *testing.T) {
	var c *ftpClient
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

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

func TestListEach_stopIteration(t *testing.T) {
	c := &fakeClient{listResult: []string{"a", "b", "c"}}
	n := 0
	err := c.ListEach(context.Background(), "/", func(e Entry) error {
		n++
		if e.Name == "b" {
			return ErrStopIteration
		}
		return nil
	})
	if err != nil || n != 2 {
		t.Fatalf("err=%v n=%d", err, n)
	}
}
