package ftp

import (
	"fmt"

	ftplib "github.com/jlaffaye/ftp"
)

// Client 对外 FTP 能力抽象（组合 Uploader / Downloader / Lister，可整体或按需 mock）。
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

// connectFTP 供单测替换；生产环境使用 defaultConnectFTP。
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

// NewClient 建立连接并登录。
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

// Close 关闭 FTP 连接。
func (c *ftpClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	err := c.conn.Quit()
	c.conn = nil
	return err
}
