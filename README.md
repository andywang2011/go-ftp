# go-ftp

Thin, testable FTP client built on [jlaffaye/ftp](https://github.com/jlaffaye/ftp).

## Install

```bash
go get github.com/andywang2011/go-ftp@v0.1.0
```

## Quick start

```go
import (
	"fmt"

	"github.com/andywang2011/go-ftp"
)

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
```

## Examples

```bash
export FTP_HOST=127.0.0.1 FTP_PORT=21 FTP_USER=u FTP_PASSWORD=p
cd go-ftp/examples/basic && go run .
cd go-ftp/examples/resume && go run .
```

## License

MIT — see [LICENSE](LICENSE).
