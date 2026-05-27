package ftp

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Uploader uploads local paths or io.Reader to remote paths.
type Uploader interface {
	Upload(ctx context.Context, localPath, remotePath string, opts ...TransferOption) error
	UploadReader(ctx context.Context, r io.Reader, remotePath string, opts ...TransferOption) error
}

var _ Uploader = (*ftpClient)(nil)

func (c *ftpClient) Upload(ctx context.Context, localPath, remotePath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return ErrNotConnected
	}
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("ftp open local: %w", err)
	}
	defer f.Close()
	return c.uploadAt(ctx, f, remotePath, localPath, opts...)
}

func (c *ftpClient) UploadReader(ctx context.Context, r io.Reader, remotePath string, opts ...TransferOption) error {
	return c.uploadAt(ctx, r, remotePath, "", opts...)
}

func (c *ftpClient) uploadAt(ctx context.Context, r io.Reader, remotePath, localPath string, opts ...TransferOption) error {
	if c == nil || c.conn == nil {
		return ErrNotConnected
	}
	to := applyTransferOpts(opts, c.transferBuf)
	plan, err := c.planUploadResume(ctx, remotePath, localPath, to, r)
	if err != nil {
		return err
	}
	if plan.done {
		return nil
	}
	if plan.offset > 0 {
		if err := seekReaderTo(r, plan.offset); err != nil {
			return err
		}
	}
	wrapped := &contextReader{ctx: ctx, r: r}
	var storErr error
	if plan.offset > 0 || to.resumeEnabled() {
		storErr = c.conn.StorFrom(remotePath, wrapped, uint64(plan.offset))
	} else {
		storErr = c.conn.Stor(remotePath, wrapped)
	}
	if storErr != nil {
		if plan.offset > 0 {
			return fmt.Errorf("ftp stor from %d: %w", plan.offset, wrapResumeNotSupported(storErr))
		}
		return fmt.Errorf("ftp stor: %w", storErr)
	}
	return nil
}
