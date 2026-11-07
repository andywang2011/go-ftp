package ftp

import (
	"errors"
	"testing"
)

func TestErrNotConnected(t *testing.T) {
	var c *ftpClient
	//lint:ignore SA1012 intentional nil context in test
	if err := c.Upload(nil, "/l", "/r"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
	//lint:ignore SA1012 intentional nil context in test
	if err := c.Download(nil, "/r", "/l"); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
	//lint:ignore SA1012 intentional nil context in test
	_, err := c.List(nil, "/")
	if !errors.Is(err, ErrNotConnected) {
		t.Fatalf("got %v", err)
	}
}
