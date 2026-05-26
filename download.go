package ftp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Downloader 下载能力（路径式 + 流式）。
type Downloader interface {
	Download(ctx context.Context, remotePath, localPath string, opts ...TransferOption) error
	DownloadWriter(ctx context.Context, remotePath string, w io.Writer, opts ...TransferOption) error
}

var _ Downloader = (*ftpClient)(nil)

func (c *ftpClient) Download(ctx context.Context, remotePath, localPath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}
	to := applyTransferOpts(opts, c.transferBuf)
	plan, err := c.planDownloadResume(ctx, remotePath, localPath, to)
	if err != nil {
		return err
	}
	if plan.done {
		return nil
	}

	var f *os.File
	if to.resumeEnabled() {
		if plan.truncateLocal {
			f, err = os.Create(localPath)
		} else {
			f, err = os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY, 0o644)
		}
		if err != nil {
			return err
		}
		if plan.offset > 0 {
			if _, err = f.Seek(plan.offset, io.SeekStart); err != nil {
				_ = f.Close()
				return err
			}
		}
	} else {
		f, err = os.Create(localPath)
		if err != nil {
			return err
		}
	}
	defer f.Close()

	if err := c.downloadAtOffset(ctx, remotePath, f, plan.offset, to); err != nil {
		return err
	}
	return c.verifyDownloadSize(ctx, remotePath, localPath, plan)
}

func (c *ftpClient) DownloadWriter(ctx context.Context, remotePath string, w io.Writer, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	to := applyTransferOpts(opts, c.transferBuf)
	plan, err := c.planDownloadResumeStream(ctx, remotePath, to, w)
	if err != nil {
		return err
	}
	if plan.done {
		return nil
	}
	return c.downloadAtOffset(ctx, remotePath, w, plan.offset, to)
}

func (c *ftpClient) downloadAtOffset(ctx context.Context, remotePath string, w io.Writer, offset int64, to transferOpts) error {
	resp, err := c.conn.RetrFrom(remotePath, uint64(offset))
	if err != nil {
		if offset > 0 {
			return fmt.Errorf("ftp retr from %d: %w", offset, wrapResumeNotSupported(err))
		}
		return fmt.Errorf("ftp retr: %w", err)
	}
	defer resp.Close()
	buf := make([]byte, to.bufferSize)
	if _, err = copyWithContext(ctx, w, resp, buf); err != nil {
		return fmt.Errorf("ftp download: %w", err)
	}
	return nil
}
