package ftp

import "errors"

var (
	// ErrNotConnected is returned when an operation runs on a nil or closed client connection.
	ErrNotConnected = errors.New("ftp: not connected")

	// ErrStopIteration may be returned from ListEach/WalkEach callbacks to stop early (not a failure).
	ErrStopIteration = errors.New("ftp: stop iteration")

	// ErrResumeNotSupported indicates the server does not support REST/SIZE or the command failed.
	ErrResumeNotSupported = errors.New("ftp: resume not supported")

	// ErrResumeOffsetMismatch means remote bytes exceed local bytes; remote partial file must be removed.
	ErrResumeOffsetMismatch = errors.New("ftp: resume offset mismatch")

	// ErrResumeNotSeekable means stream resume requires io.Seeker on the reader or writer.
	ErrResumeNotSeekable = errors.New("ftp: resume requires io.Seeker")
)
