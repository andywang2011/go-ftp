package ftp

import (
	"errors"
	"testing"
)

func TestErrNotConnected(t *testing.T) {
	var c *ftpClient
	if err := c.Upload(nil, "/l", "/r"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
	if err := c.Download(nil, "/r", "/l"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
	_, err := c.List(nil, "/")
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
}
