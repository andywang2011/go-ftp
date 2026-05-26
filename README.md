# go-ftp

FTP 客户端：`Client` 接口、流式大文件传输、大目录逐条列举。

```bash
go get github.com/andywang2011/go-ftp@v0.1.0
```

## 大文件 Upload / Download

| 场景 | 推荐 API |
|------|----------|
| 本地路径 ↔ 远程路径 | `Upload` / `Download`（内部流式，不整文件读入内存） |
| 已有 `io.Reader` / `io.Writer`（管道、对象存储中转） | `UploadReader` / `DownloadWriter` |
| 可取消、调缓冲区 | `TransferOption`：`WithBufferSize(512<<10)` |

```go
// 1GB+ 建议增大缓冲区，并用 context 控制超时/取消
ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
defer cancel()

f, _ := os.Open("/data/big.bin")
defer f.Close()
err := cli.UploadReader(ctx, f, "/remote/big.bin", ftp.WithBufferSize(1024*1024))

out, _ := os.Create("/data/out.bin")
defer out.Close()
err = cli.DownloadWriter(ctx, "/remote/big.bin", out, ftp.WithBufferSize(1024*1024))
```

`Config.TransferBufferSize` 为全局默认（默认 **256KiB**）。

## 大文件断点续传

依赖服务端 **REST**（续传偏移）与 **SIZE**（远程文件大小），常见 vsftpd / ProFTPD 可用。

| 选项 | 作用 |
|------|------|
| `WithResume(true)` | 路径式 `Upload` / `Download`：按本地/远程已有字节自动续传 |
| `WithResumeOffset(n)` | 显式从第 `n` 字节继续；流式须 `io.Seeker` |

```go
// 下载中断后再次调用，从本地 partial 文件续传
err := cli.Download(ctx, "/remote/big.bin", "/data/big.bin", ftp.WithResume(true))

// 上传：远程已有部分字节时，从 FileSize(remote) 对齐后续传
err := cli.Upload(ctx, "/data/big.bin", "/remote/big.bin", ftp.WithResume(true))
```

错误处理：

- `ErrResumeNotSupported` — 服务端不支持或 REST/SIZE 失败
- `ErrResumeOffsetMismatch` — 远程已传大于本地，需删除远程后重传
- `ErrResumeNotSeekable` — 流式续传时 `Reader`/`Writer` 未实现 `io.Seeker`

本地文件大于远程时，下载会**截断本地**后全量重下（远程文件可能被替换）。

## 目录里文件很多怎么 List？

FTP **没有**标准的服务端分页；一次 `LIST` 通常会返回整个目录。

| 规模 | 推荐 |
|------|------|
| 少量文件（< 几百） | `List` → `[]string` |
| 较多、需边处理边释放 | `ListEach(dir, func(Entry) error { ... })`，处理完一条即可丢弃 |
| 需提前结束 | 回调里 `return ftp.ErrStopIteration` |
| 子树很多 | `WalkEach(root, fn)` 按 Walk 顺序逐条处理 |
| 习惯 `for` + `Next` | `ListIter`（内部仍一次 LIST，但调用方不持有大切片） |

```go
// 只处理前 100 个 regular file
count := 0
err := cli.ListEach(ctx, "/inbox", func(e Entry) error {
    if e.Type != ftp.EntryTypeFile {
        return nil
    }
    count++
    if count >= 100 {
        return ftp.ErrStopIteration
    }
    return process(e)
})
```

超大规模（十万级单目录）建议在**业务层拆分子目录**或按文件名规则分桶，不要依赖单次 `List`。

## Client 接口

`Client` = `Uploader` + `Downloader` + `Lister` + `Close()`，定义分散在 `upload.go` / `download.go` / `list.go`。业务可依赖完整 `Client`，或只注入窄接口：

```go
func NewUploadJob(u ftp.Uploader) *UploadJob { return &UploadJob{u: u} }
func NewSyncJob(l ftp.Lister) *SyncJob       { return &SyncJob{l: l} }
```

凭证由应用 viper 注入，库内**不**设置 anonymous 默认账号。

## 生产 / 开源说明

- **连接**：`NewClient` 后务必 `defer cli.Close()`；用 `context` 控制传输超时。
- **传输**：当前为明文 FTP（端口 21）；**未实现 FTPS/TLS**，勿在公网传敏感数据。
- **续传**：依赖服务端 `REST`/`SIZE`；上传前若 `ErrResumeOffsetMismatch`，需清理远程半文件。
- **下载失败**：未开 `WithResume` 时，失败可能留下不完整本地文件；大文件建议 `WithResume(true)`。
- **并发**：单 `Client` 同一时刻仅支持一个数据连接（底层库限制），勿并发传输。
- **版本**：发版请打 git tag（如 `v0.2.0`），与 `go get ...@v0.2.0` 一致。

## 可运行示例

见 [examples/basic/main.go](examples/basic/main.go)：

```bash
export FTP_HOST=127.0.0.1 FTP_PORT=21 FTP_USER=u FTP_PASSWORD=p
cd go-ftp && go run ./examples/basic
```
