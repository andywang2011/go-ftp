package ftp

import (
	"io"

	ftplib "github.com/jlaffaye/ftp"
)

// dirWalker 目录遍历（与 jlaffaye/ftp.Walker 对齐，便于单测 mock）。
type dirWalker interface {
	Next() bool
	Err() error
	Stat() *ftplib.Entry
	Path() string
}

// ftpConn 传输与列举所需的最小 FTP 连接能力（便于单测 mock）。
type ftpConn interface {
	RetrFrom(path string, offset uint64) (io.ReadCloser, error)
	Stor(path string, r io.Reader) error
	StorFrom(path string, r io.Reader, offset uint64) error
	FileSize(path string) (int64, error)
	List(path string) ([]*ftplib.Entry, error)
	Walk(root string) dirWalker
	Quit() error
}

type jlaffayeConn struct {
	*ftplib.ServerConn
}

func (c *jlaffayeConn) RetrFrom(path string, offset uint64) (io.ReadCloser, error) {
	return c.ServerConn.RetrFrom(path, offset)
}

func (c *jlaffayeConn) Walk(root string) dirWalker {
	return c.ServerConn.Walk(root)
}

var _ ftpConn = (*jlaffayeConn)(nil)
