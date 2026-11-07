package ftp_test

import (
	"fmt"

	"github.com/andywang2011/go-ftp"
)

func ExampleNewClient() {
	cfg := ftp.ConfigDefault()
	cfg.Host = "127.0.0.1"
	cfg.Port = 65530
	cfg.User = "user"
	cfg.Password = "secret"

	cli, err := ftp.NewClient(cfg)
	if err != nil {
		fmt.Println("connect failed")
		return
	}
	defer cli.Close()
	fmt.Println("ok")
	// Output: connect failed
}
