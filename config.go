package ftp

import (
	"errors"
	"strings"
	"time"
)

// Config FTP 连接配置（用户名/密码由引入方注入，库内不设 anonymous 默认值）。
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Timeout            time.Duration
	TLS                bool
	TransferBufferSize int // 大文件流式传输缓冲区，默认 256KiB
}

const (
	defaultPort           = 21
	defaultTransferBuffer = 256 << 10
)

// Validate 校验必填项并填充端口、超时默认值。
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return errors.New("ftp: host is required")
	}
	if strings.TrimSpace(c.User) == "" {
		return errors.New("ftp: user is required")
	}
	if c.Password == "" {
		return errors.New("ftp: password is required")
	}
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second
	}
	if c.TransferBufferSize == 0 {
		c.TransferBufferSize = defaultTransferBuffer
	}
	return nil
}

func (c Config) transferBufferSize() int {
	if c.TransferBufferSize > 0 {
		return c.TransferBufferSize
	}
	return defaultTransferBuffer
}

// Addr 返回 host:port。
func (c Config) Addr() string {
	return c.Host + ":" + itoa(c.Port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
