package ftp

import (
	"errors"
	"strconv"
	"time"
)

// Config holds FTP connection settings. Credentials must be set by the caller (no anonymous default).
type Config struct {
	// Host is the server hostname or IP (required).
	Host string
	// Port is the control port (default 21). FTPS/TLS is not implemented.
	Port int
	// User is the login name (required).
	User string
	// Password is the login password (required).
	Password string
	// Timeout is used for dial and control connection operations (default 30s).
	Timeout time.Duration
	// TransferBufferSize is the default io buffer for uploads/downloads (default 256KiB).
	TransferBufferSize int
}

const (
	defaultPort           = 21
	defaultTransferBuffer = 256 << 10
)

// ConfigDefault returns port, timeout, and buffer defaults only. Host, User, and Password must be set by the caller.
func ConfigDefault() Config {
	return Config{
		Port:               defaultPort,
		Timeout:            30 * time.Second,
		TransferBufferSize: defaultTransferBuffer,
	}
}

// Validate checks required fields and fills defaults for port, timeout, and buffer size.
func (c *Config) Validate() error {
	if c.Host == "" {
		return errors.New("ftp: host is required")
	}
	if c.User == "" {
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

// Addr returns host:port (call Validate first to apply default port).
func (c Config) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}
