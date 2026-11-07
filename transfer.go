package ftp

import (
	"context"
	"errors"
	"io"
)

// TransferOption configures a single upload or download.
type TransferOption func(*transferOpts)

type transferOpts struct {
	bufferSize           int
	resumeAuto           bool
	resumeOffset         int64
	resumeOffsetExplicit bool
}

func (o transferOpts) resumeEnabled() bool {
	return o.resumeAuto || o.resumeOffsetExplicit
}

// WithResume enables automatic offset detection for path-based transfers (requires REST/SIZE).
func WithResume(enabled bool) TransferOption {
	return func(o *transferOpts) {
		o.resumeAuto = enabled
	}
}

// WithResumeOffset sets an explicit start byte; streaming transfers require io.Seeker.
func WithResumeOffset(n int64) TransferOption {
	return func(o *transferOpts) {
		if n >= 0 {
			o.resumeOffset = n
			o.resumeOffsetExplicit = true
		}
	}
}

// WithBufferSize sets the copy buffer for one transfer (overrides Config.TransferBufferSize).
func WithBufferSize(n int) TransferOption {
	return func(o *transferOpts) {
		if n > 0 {
			o.bufferSize = n
		}
	}
}

func applyTransferOpts(opts []TransferOption, defaultBuf int) transferOpts {
	o := transferOpts{bufferSize: defaultBuf}
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	if o.bufferSize <= 0 {
		o.bufferSize = defaultTransferBuffer
	}
	return o
}

// contextReader checks ctx on each Read so large uploads can be cancelled mid-stream.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *contextReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, defaultTransferBuffer)
	}
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("ftp: invalid write count")
				}
			}
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}
