// 基本用法示例：连接 FTP，上传/下载、大目录列举。
//
// 运行前设置环境变量（勿把密码写进代码）：
//
//	export FTP_HOST=127.0.0.1
//	export FTP_PORT=21
//	export FTP_USER=your_user
//	export FTP_PASSWORD=your_pass
//
// 在 go-ftp 目录执行：
//
//	go run ./examples/basic
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
		log.Fatal("new client:", err)
	}
	defer func() {
		if err := cli.Close(); err != nil {
			log.Println("close:", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := runExamples(ctx, cli); err != nil {
		log.Fatal(err)
	}
	log.Println("done")
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
			return ftp.Config{}, fmt.Errorf("FTP_PORT: %w", err)
		}
		port = n
	}
	return ftp.Config{
		Host:               host,
		Port:               port,
		User:               user,
		Password:           pass,
		Timeout:            30 * time.Second,
		TransferBufferSize: 256 << 10,
	}, nil
}

func runExamples(ctx context.Context, cli ftp.Client) error {
	const remoteDir = "/"
	const remoteFile = "/hello-from-go-ftp.txt"
	localFile := filepath.Join(os.TempDir(), "go-ftp-demo-local.txt")

	// --- 1. 小文件：路径上传 ---
	if err := os.WriteFile(localFile, []byte("hello ftp\n"), 0o644); err != nil {
		return err
	}
	log.Println("upload:", localFile, "->", remoteFile)
	if err := cli.Upload(ctx, localFile, remoteFile); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// --- 2. 路径下载；大文件中断后可 WithResume(true) 续传 ---
	downloaded := localFile + ".downloaded"
	log.Println("download:", remoteFile, "->", downloaded)
	if err := cli.Download(ctx, remoteFile, downloaded,
		ftp.WithBufferSize(512<<10),
		ftp.WithResume(true), // 若 downloaded 已存在 partial，从本地大小续传
	); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// --- 3. 流式上传（内存 -> 远程）---
	remoteFromMem := "/from-memory.txt"
	log.Println("upload reader ->", remoteFromMem)
	if err := cli.UploadReader(ctx, strings.NewReader("stream upload\n"), remoteFromMem); err != nil {
		return fmt.Errorf("upload reader: %w", err)
	}

	// --- 4. 目录文件少：List ---
	names, err := cli.List(ctx, remoteDir)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	log.Printf("list %s: %d entries (show up to 5)", remoteDir, len(names))
	for i, name := range names {
		if i >= 5 {
			break
		}
		log.Println("  -", name)
	}

	// --- 5. 目录文件多：ListEach（边处理边丢，可提前结束）---
	var fileCount int
	log.Println("list each:", remoteDir)
	err = cli.ListEach(ctx, remoteDir, func(e ftp.Entry) error {
		if e.Type != ftp.EntryTypeFile {
			return nil
		}
		fileCount++
		if fileCount > 20 {
			return ftp.ErrStopIteration
		}
		log.Printf("  file: name=%s size=%d", e.Name, e.Size)
		return nil
	})
	if err != nil {
		return fmt.Errorf("list each: %w", err)
	}

	// --- 6. 子树遍历：WalkEach（路径在 e.Name）---
	log.Println("walk:", remoteDir)
	walkCount := 0
	err = cli.WalkEach(ctx, remoteDir, func(e ftp.Entry) error {
		walkCount++
		if walkCount > 10 {
			return ftp.ErrStopIteration
		}
		log.Printf("  walk: path=%s type=%d size=%d", e.Name, e.Type, e.Size)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk: %w", err)
	}

	return nil
}
