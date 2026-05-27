# go-ftp

Thin, testable FTP client for Go, built on [jlaffaye/ftp](https://github.com/jlaffaye/ftp).

```bash
go get github.com/andywang2011/go-ftp@v0.1.0
```

## Design

- No viper, no global singletons, no `log.Fatal` inside the library.
- `Client` = `Uploader` + `Downloader` + `Lister` + `Close()` (inject narrow interfaces in tests).
- Credentials come from your app; there is no anonymous default user.
- Plain FTP only; **FTPS/TLS is not implemented**.

## Quick start

```go
cfg := ftp.Config{
    Host:     "127.0.0.1",
    Port:     21,
    User:     "user",
    Password: "secret",
}
cli, err := ftp.NewClient(cfg)
if err != nil { /* handle */ }
defer cli.Close()

ctx := context.Background()
if err := cli.Upload(ctx, "/local/file.bin", "/remote/file.bin"); err != nil {
    /* handle */
}
```

`ConfigDefault()` sets port (21), timeout (30s), and transfer buffer (256KiB) only—you must still set `Host`, `User`, and `Password`.

## Large file upload / download

| Scenario | API |
|----------|-----|
| Local path ↔ remote path | `Upload` / `Download` (streaming, not full-file RAM) |
| `io.Reader` / `io.Writer` | `UploadReader` / `DownloadWriter` |
| Per-transfer buffer | `WithBufferSize(512 << 10)` |
| Default buffer | `Config.TransferBufferSize` (256KiB) |

Use `context.WithTimeout` or cancellation for long transfers.

## Resume (REST / SIZE)

| Option | Effect |
|--------|--------|
| `WithResume(true)` | Path-based: align local/remote partial sizes |
| `WithResumeOffset(n)` | Explicit offset; streams need `io.Seeker` |

Errors: `ErrResumeNotSupported`, `ErrResumeOffsetMismatch`, `ErrResumeNotSeekable`.

See [examples/resume](examples/resume/main.go).

## Listing large directories

FTP has no standard server-side pagination; one `LIST` usually returns the whole directory.

| Scale | API |
|-------|-----|
| Small (< few hundred) | `List` → `[]string` |
| Process without holding all names | `ListEach` |
| Stop early | return `ErrStopIteration` from callback |
| Directory tree | `WalkEach` (`Entry.Name` is full path) |
| Iterator style | `ListIter` (method returns type `ListIter`; holds entries after one LIST) |

For very large single directories, split paths in your application.

## Configuration

| Field | Description |
|-------|-------------|
| `Host` | Server hostname (required) |
| `Port` | Control port (default 21) |
| `User` / `Password` | Credentials (required) |
| `Timeout` | Dial / control timeout (default 30s) |
| `TransferBufferSize` | Default copy buffer (default 256KiB) |

## Production notes

- Always `defer cli.Close()` after `NewClient`.
- Do not run concurrent transfers on one `Client` (single data connection).
- Do not use plain FTP over untrusted networks for sensitive data.
- Tag releases (e.g. `v0.1.0`) for `go get ...@version`.

## Examples

```bash
export FTP_HOST=127.0.0.1 FTP_PORT=21 FTP_USER=u FTP_PASSWORD=p
cd go-ftp && go run ./examples/basic
cd go-ftp/examples/resume && go run .
```

## 中文摘要

- 库内无 viper/全局单例；凭证由业务注入。
- 大文件流式传输；`ListEach`/`WalkEach` 适合大目录。
- 断点续传依赖服务端 REST/SIZE；明文 FTP，无 FTPS。

## License

MIT — see [LICENSE](LICENSE).
