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

对外只暴露一个 `Client`；业务与单测 mock 依赖它即可。若某模块只需上传，可在**业务包**里自建窄接口，例如：

```go
type FTPUploader interface {
    UploadReader(ctx context.Context, r io.Reader, remote string, opts ...ftp.TransferOption) error
}
func NewJob(cli ftp.Client) *Job { return &Job{cli: cli} } // cli 仍满足窄接口
```

凭证由应用 viper 注入，库内**不**设置 anonymous 默认账号。

## 可运行示例

见 [examples/basic/main.go](examples/basic/main.go)：

```bash
export FTP_HOST=127.0.0.1 FTP_PORT=21 FTP_USER=u FTP_PASSWORD=p
cd go-ftp && go run ./examples/basic
```
