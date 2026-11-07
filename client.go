package ftp

import (
	"fmt"

	ftplib "github.com/jlaffaye/ftp"
)

// Client is the full FTP capability set (upload, download, list, close). Use narrow interfaces when mocking.
type Client interface {
	Uploader
	Downloader
	Lister
	Close() error
}

var _ Client = (*ftpClient)(nil)

type ftpClient struct {
	conn        ftpConn
	transferBuf int
}

// connectFTP is replaced in tests; production uses defaultConnectFTP.
var connectFTP = defaultConnectFTP

func defaultConnectFTP(cfg Config) (ftpConn, error) {
	conn, err := ftplib.Dial(cfg.Addr(), ftplib.DialWithTimeout(cfg.Timeout))
	if err != nil {
		return nil, fmt.Errorf("ftp dial: %w", err)
	}
	if err = conn.Login(cfg.User, cfg.Password); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("ftp login: %w", err)
	}
	return &jlaffayeConn{conn}, nil
}

// NewClient dials, logs in, and returns a ready Client. Caller must Close when done.
func NewClient(cfg Config) (Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	conn, err := connectFTP(cfg)
	if err != nil {
		return nil, err
	}
	return &ftpClient{conn: conn, transferBuf: cfg.transferBufferSize()}, nil
}

// Close sends QUIT and releases the connection. Safe on nil receiver.
func (c *ftpClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	err := c.conn.Quit()
	c.conn = nil
	return err
}
