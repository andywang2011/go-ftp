package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/textproto"
	"os"

	ftplib "github.com/jlaffaye/ftp"
)

var (
	// ErrResumeNotSupported 服务端不支持 REST/SIZE 或续传命令失败。
	ErrResumeNotSupported = errors.New("ftp: resume not supported")
	// ErrResumeOffsetMismatch 远程已传字节大于本地，无法安全续传。
	ErrResumeOffsetMismatch = errors.New("ftp: resume offset mismatch")
	// ErrResumeNotSeekable 流式续传要求 io.Seeker。
	ErrResumeNotSeekable = errors.New("ftp: resume requires io.Seeker")
)

type resumePlan struct {
	offset        int64
	remoteSize    int64
	done          bool
	truncateLocal bool
}

func isFTPFileUnavailable(err error) bool {
	var te *textproto.Error
	if errors.As(err, &te) {
		return te.Code == ftplib.StatusFileUnavailable
	}
	return false
}

func wrapResumeNotSupported(err error) error {
	if err == nil {
		return ErrResumeNotSupported
	}
	return fmt.Errorf("%w: %w", ErrResumeNotSupported, err)
}

func (c *ftpClient) planDownloadResume(ctx context.Context, remote, local string, to transferOpts) (resumePlan, error) {
	if err := ctx.Err(); err != nil {
		return resumePlan{}, err
	}
	if !to.resumeEnabled() {
		return resumePlan{}, nil
	}

	var localOff int64
	switch {
	case to.resumeOffsetExplicit:
		localOff = to.resumeOffset
	case to.resumeAuto && local != "":
		st, err := os.Stat(local)
		if err != nil {
			if !os.IsNotExist(err) {
				return resumePlan{}, err
			}
		} else {
			localOff = st.Size()
		}
	default:
		return resumePlan{}, nil
	}

	remoteSize, err := c.conn.FileSize(remote)
	if err != nil {
		return resumePlan{}, wrapResumeNotSupported(fmt.Errorf("ftp size %s: %w", remote, err))
	}

	if localOff > remoteSize {
		return resumePlan{remoteSize: remoteSize, truncateLocal: true}, nil
	}
	if localOff >= remoteSize {
		return resumePlan{offset: localOff, remoteSize: remoteSize, done: true}, nil
	}
	return resumePlan{offset: localOff, remoteSize: remoteSize}, nil
}

func (c *ftpClient) planDownloadResumeStream(ctx context.Context, remote string, to transferOpts, w io.Writer) (resumePlan, error) {
	if err := ctx.Err(); err != nil {
		return resumePlan{}, err
	}
	if !to.resumeEnabled() {
		return resumePlan{}, nil
	}
	if to.resumeAuto && !to.resumeOffsetExplicit {
		return resumePlan{}, ErrResumeNotSeekable
	}
	if !to.resumeOffsetExplicit {
		return resumePlan{}, nil
	}

	seeker, ok := w.(io.Seeker)
	if !ok {
		return resumePlan{}, ErrResumeNotSeekable
	}
	if _, err := seeker.Seek(to.resumeOffset, io.SeekStart); err != nil {
		return resumePlan{}, fmt.Errorf("ftp seek writer: %w", err)
	}
	localOff := to.resumeOffset

	remoteSize, err := c.conn.FileSize(remote)
	if err != nil {
		return resumePlan{}, wrapResumeNotSupported(fmt.Errorf("ftp size %s: %w", remote, err))
	}
	if localOff > remoteSize {
		if _, err = seeker.Seek(0, io.SeekStart); err != nil {
			return resumePlan{}, fmt.Errorf("ftp seek writer: %w", err)
		}
		return resumePlan{remoteSize: remoteSize}, nil
	}
	if localOff >= remoteSize {
		return resumePlan{offset: localOff, remoteSize: remoteSize, done: true}, nil
	}
	return resumePlan{offset: localOff, remoteSize: remoteSize}, nil
}

func (c *ftpClient) planUploadResume(ctx context.Context, remote, localPath string, to transferOpts, r io.Reader) (resumePlan, error) {
	if err := ctx.Err(); err != nil {
		return resumePlan{}, err
	}
	if !to.resumeEnabled() {
		return resumePlan{}, nil
	}

	localSize, err := localFileSize(localPath, r, to)
	if err != nil {
		return resumePlan{}, err
	}

	remoteOff, err := c.remoteUploadedSize(remote)
	if err != nil {
		return resumePlan{}, err
	}
	if remoteOff > localSize {
		return resumePlan{}, fmt.Errorf("%w: remote=%d local=%d", ErrResumeOffsetMismatch, remoteOff, localSize)
	}
	if remoteOff == localSize {
		return resumePlan{offset: remoteOff, remoteSize: localSize, done: true}, nil
	}

	if localPath == "" {
		if err := seekReaderTo(r, remoteOff); err != nil {
			return resumePlan{}, err
		}
	}
	return resumePlan{offset: remoteOff, remoteSize: localSize}, nil
}

func localFileSize(localPath string, r io.Reader, to transferOpts) (int64, error) {
	if localPath != "" {
		st, err := os.Stat(localPath)
		if err != nil {
			return 0, err
		}
		return st.Size(), nil
	}
	if !to.resumeEnabled() {
		return 0, nil
	}
	seeker, ok := r.(io.Seeker)
	if !ok {
		return 0, ErrResumeNotSeekable
	}
	end, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("ftp seek reader: %w", err)
	}
	return end, nil
}

func seekReaderTo(r io.Reader, off int64) error {
	seeker, ok := r.(io.Seeker)
	if !ok {
		return ErrResumeNotSeekable
	}
	if _, err := seeker.Seek(off, io.SeekStart); err != nil {
		return fmt.Errorf("ftp seek reader: %w", err)
	}
	return nil
}

func (c *ftpClient) remoteUploadedSize(remote string) (int64, error) {
	size, err := c.conn.FileSize(remote)
	if err != nil {
		if isFTPFileUnavailable(err) {
			return 0, nil
		}
		return 0, wrapResumeNotSupported(fmt.Errorf("ftp size %s: %w", remote, err))
	}
	return size, nil
}

func (c *ftpClient) verifyDownloadSize(ctx context.Context, remote, local string, plan resumePlan) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if local == "" || plan.remoteSize <= 0 {
		return nil
	}
	st, err := os.Stat(local)
	if err != nil {
		return err
	}
	if st.Size() != plan.remoteSize {
		remoteSize, err := c.conn.FileSize(remote)
		if err != nil {
			return fmt.Errorf("ftp download: size mismatch local=%d expected=%d", st.Size(), plan.remoteSize)
		}
		if st.Size() != remoteSize {
			return fmt.Errorf("ftp download: size mismatch local=%d remote=%d", st.Size(), remoteSize)
		}
	}
	return nil
}
