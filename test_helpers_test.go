package ftp

import (
	"errors"
	"io"

	ftplib "github.com/jlaffaye/ftp"
)

type bytesSink struct{ b []byte }

func (s *bytesSink) Write(p []byte) (int, error) {
	s.b = append(s.b, p...)
	return len(p), nil
}

type brokenSeeker struct{ io.Reader }

func (brokenSeeker) Write(p []byte) (int, error) { return len(p), nil }
func (brokenSeeker) Seek(int64, int) (int64, error) {
	return 0, errors.New("seek fail")
}

type invalidCountWriter struct{}

func (invalidCountWriter) Write([]byte) (int, error) { return -1, nil }

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) { return 0, nil }

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }

type readOnly struct{ io.Reader }

type nilWalkerConn struct {
	fake *fakeFTPConn
}

func (n *nilWalkerConn) RetrFrom(path string, offset uint64) (io.ReadCloser, error) {
	return n.fake.RetrFrom(path, offset)
}
func (n *nilWalkerConn) Stor(path string, r io.Reader) error { return n.fake.Stor(path, r) }
func (n *nilWalkerConn) StorFrom(path string, r io.Reader, offset uint64) error {
	return n.fake.StorFrom(path, r, offset)
}
func (n *nilWalkerConn) FileSize(path string) (int64, error)       { return n.fake.FileSize(path) }
func (n *nilWalkerConn) List(path string) ([]*ftplib.Entry, error) { return n.fake.List(path) }
func (n *nilWalkerConn) Walk(string) dirWalker                     { return nil }
func (n *nilWalkerConn) Quit() error                               { return n.fake.Quit() }
