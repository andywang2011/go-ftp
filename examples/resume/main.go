// Resume example: path-based download with WithResume(true).
//
//	export FTP_HOST=127.0.0.1 FTP_USER=u FTP_PASSWORD=p
//	go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/andywang2011/go-ftp"
)

func main() {
	cfg, err := configFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	cli, err := ftp.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	remote := "/large-file.bin"
	local := filepath.Join(os.TempDir(), "go-ftp-resume-demo.bin")

	log.Println("download with resume:", remote, "->", local)
	if err := cli.Download(ctx, remote, local, ftp.WithResume(true), ftp.WithBufferSize(1024*1024)); err != nil {
		log.Fatal(err)
	}
	log.Println("complete, size:", fileSize(local))
}

func configFromEnv() (ftp.Config, error) {
	host := os.Getenv("FTP_HOST")
	user := os.Getenv("FTP_USER")
	pass := os.Getenv("FTP_PASSWORD")
	if host == "" || user == "" || pass == "" {
		return ftp.Config{}, fmt.Errorf("need FTP_HOST, FTP_USER, FTP_PASSWORD")
	}
	port := 21
	if p := os.Getenv("FTP_PORT"); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return ftp.Config{}, err
		}
		port = n
	}
	return ftp.Config{
		Host: host, Port: port, User: user, Password: pass,
		Timeout: 30 * time.Second, TransferBufferSize: 256 << 10,
	}, nil
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return st.Size()
}
