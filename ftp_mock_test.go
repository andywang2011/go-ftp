package ftp

import (
	"fmt"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type ftpMock struct {
	t        *testing.T
	listener net.Listener
	port     int
	mu       sync.Mutex
	conns    []net.Conn
	wg       sync.WaitGroup
}

func startFTPMock(t *testing.T) (*ftpMock, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &ftpMock{t: t, listener: ln, port: ln.Addr().(*net.TCPAddr).Port}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.serve()
	}()
	return m, m.port
}

// Close stops the listener, closes every accepted control connection and waits
// for handler goroutines so goleak sees no stragglers.
func (m *ftpMock) Close() {
	_ = m.listener.Close()
	m.mu.Lock()
	for _, c := range m.conns {
		_ = c.Close()
	}
	m.mu.Unlock()
	m.wg.Wait()
}

func (m *ftpMock) track(c net.Conn) {
	m.mu.Lock()
	m.conns = append(m.conns, c)
	m.mu.Unlock()
}

func (m *ftpMock) serve() {
	for {
		conn, err := m.listener.Accept()
		if err != nil {
			return
		}
		m.track(conn)
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.handle(conn)
		}()
	}
}

func (m *ftpMock) handle(conn net.Conn) {
	defer conn.Close()
	tc := textproto.NewConn(conn)
	_ = tc.PrintfLine("220 mock FTP ready")
	var rest int64
	for {
		line, err := tc.ReadLine()
		if err != nil {
			return
		}
		parts := strings.SplitN(line, " ", 2)
		cmd := parts[0]
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		switch cmd {
		case "FEAT":
			_ = tc.PrintfLine("211-Features:\r\n SIZE\r\n REST\r\n211 End")
		case "USER":
			_ = tc.PrintfLine("331 password")
		case "PASS":
			_ = tc.PrintfLine("230 logged in")
		case "TYPE":
			_ = tc.PrintfLine("200 type ok")
		case "OPTS":
			_ = tc.PrintfLine("200 ok")
		case "PWD":
			_ = tc.PrintfLine("257 \"/\"")
		case "SIZE":
			if arg == "/magic-file" {
				_ = tc.PrintfLine("213 42")
			} else {
				_ = tc.PrintfLine("550 unavailable")
			}
		case "REST":
			off, _ := strconv.ParseInt(arg, 10, 64)
			rest = off
			_ = tc.PrintfLine("350 restart")
		case "PASV":
			p2, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				_ = tc.PrintfLine("425 fail")
				continue
			}
			addr := p2.Addr().(*net.TCPAddr)
			p1 := addr.Port / 256
			p0 := addr.Port % 256
			_ = tc.PrintfLine("227 Entering Passive Mode (127,0,0,1,%d,%d).", p1, p0)
			go func() {
				defer p2.Close()
				dc, err := p2.Accept()
				if err != nil {
					return
				}
				defer dc.Close()
				payload := []byte("01234567890123456789012345678901234567890") // 42 bytes
				if int(rest) < len(payload) {
					_, _ = dc.Write(payload[rest:])
				}
			}()
		case "RETR":
			_ = tc.PrintfLine("150 opening")
			_ = tc.PrintfLine("226 done")
		case "QUIT":
			_ = tc.PrintfLine("221 bye")
			return
		default:
			_ = tc.PrintfLine("502 %s", cmd)
		}
	}
}

func startFTPMockLoginFail(t *testing.T) (*ftpMock, int) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	m := &ftpMock{t: t, listener: ln, port: ln.Addr().(*net.TCPAddr).Port}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.serveLoginFail()
	}()
	return m, m.port
}

func (m *ftpMock) serveLoginFail() {
	conn, err := m.listener.Accept()
	if err != nil {
		return
	}
	defer m.listener.Close()
	defer conn.Close()
	tc := textproto.NewConn(conn)
	_ = tc.PrintfLine("220 mock")
	for {
		line, err := tc.ReadLine()
		if err != nil {
			return
		}
		switch strings.SplitN(line, " ", 2)[0] {
		case "FEAT":
			_ = tc.PrintfLine("211 End")
		case "USER":
			_ = tc.PrintfLine("331 pass")
		case "PASS":
			_ = tc.PrintfLine("530 login failed")
			return
		case "QUIT":
			return
		default:
			_ = tc.PrintfLine("200 ok")
		}
	}
}

func TestFTPMock_port(t *testing.T) {
	m, port := startFTPMock(t)
	defer m.Close()
	if port <= 0 {
		t.Fatal(port)
	}
	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	_ = c.Close()
}
