package ftp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	ftplib "github.com/jlaffaye/ftp"
)

// Client 对外 FTP 能力抽象（窄接口 + 可 mock + 大文件流式）。
type Client interface {
	// 路径式：内部走流式，适合常规文件。
	Upload(ctx context.Context, localPath, remotePath string, opts ...TransferOption) error
	Download(ctx context.Context, remotePath, localPath string, opts ...TransferOption) error

	// 流式：大文件 / 管道 / 不落盘场景优先使用。
	UploadReader(ctx context.Context, r io.Reader, remotePath string, opts ...TransferOption) error
	DownloadWriter(ctx context.Context, remotePath string, w io.Writer, opts ...TransferOption) error

	// 小目录：一次返回全部文件名。
	List(ctx context.Context, remoteDir string) ([]string, error)

	// 大目录：逐条回调，可 ErrStopIteration 提前结束。
	ListEach(ctx context.Context, remoteDir string, fn func(Entry) error) error
	ListIter(ctx context.Context, remoteDir string) (*ListIter, error)
	WalkEach(ctx context.Context, root string, fn func(Entry) error) error

	Close() error
}

var _ Client = (*ftpClient)(nil)

type ftpClient struct {
	conn        *ftplib.ServerConn
	transferBuf int
}

// NewClient 建立连接并登录。
func NewClient(cfg Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	conn, err := ftplib.Dial(cfg.Addr(), ftplib.DialWithTimeout(cfg.Timeout))
	if err != nil {
		return nil, fmt.Errorf("ftp dial: %w", err)
	}
	if err = conn.Login(cfg.User, cfg.Password); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("ftp login: %w", err)
	}
	return &ftpClient{conn: conn, transferBuf: cfg.transferBufferSize()}, nil
}

func (c *ftpClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	err := c.conn.Quit()
	c.conn = nil
	return err
}

func (c *ftpClient) Upload(ctx context.Context, localPath, remotePath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("ftp open local: %w", err)
	}
	defer f.Close()
	return c.UploadReader(ctx, f, remotePath, opts...)
}

func (c *ftpClient) UploadReader(ctx context.Context, r io.Reader, remotePath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	to := applyTransferOpts(opts, c.transferBuf)
	wrapped := &contextReader{ctx: ctx, r: r}
	if err := c.conn.Stor(remotePath, wrapped); err != nil {
		return fmt.Errorf("ftp stor: %w", err)
	}
	_ = to // Stor 由服务端读尽 reader；缓冲区由底层连接控制
	return nil
}

func (c *ftpClient) Download(ctx context.Context, remotePath, localPath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.DownloadWriter(ctx, remotePath, f, opts...)
}

func (c *ftpClient) DownloadWriter(ctx context.Context, remotePath string, w io.Writer, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	to := applyTransferOpts(opts, c.transferBuf)
	resp, err := c.conn.Retr(remotePath)
	if err != nil {
		return fmt.Errorf("ftp retr: %w", err)
	}
	defer resp.Close()
	buf := make([]byte, to.bufferSize)
	if _, err = copyWithContext(ctx, w, resp, buf); err != nil {
		return fmt.Errorf("ftp download: %w", err)
	}
	return nil
}

func (c *ftpClient) List(ctx context.Context, remoteDir string) ([]string, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("ftp: not connected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := c.conn.List(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("ftp list: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names, nil
}
