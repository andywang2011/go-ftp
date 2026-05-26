package ftp

import (
	"bytes"
	"io"
	"net/textproto"
	"sync"

	ftplib "github.com/jlaffaye/ftp"
)

type fakeFTPConn struct {
	mu sync.Mutex

	fileSizes   map[string]int64
	retrOffsets []uint64
	storOffsets []uint64
	storData    []byte
	listEntries []*ftplib.Entry
	walk        *fakeWalker

	retrErr     error
	storErr     error
	listErr     error
	quitErr     error
	fileSizeErr map[string]error
}

type fakeWalker struct {
	entries []*ftplib.Entry
	paths   []string
	idx     int // starts at -1; first Next() moves to 0
	errAt   int
	walkErr error
}

func (w *fakeWalker) Next() bool {
	if w == nil {
		return false
	}
	w.idx++
	return w.idx < len(w.paths)
}

func (w *fakeWalker) Err() error {
	if w != nil && w.errAt >= 0 && w.idx == w.errAt && w.walkErr != nil {
		return w.walkErr
	}
	return nil
}

func (w *fakeWalker) Stat() *ftplib.Entry {
	if w == nil || w.idx < 0 || w.idx >= len(w.entries) {
		return nil
	}
	return w.entries[w.idx]
}

func (w *fakeWalker) Path() string {
	if w == nil || w.idx < 0 || w.idx >= len(w.paths) {
		return ""
	}
	return w.paths[w.idx]
}

func newFakeFTPConn() *fakeFTPConn {
	return &fakeFTPConn{
		fileSizes:   make(map[string]int64),
		fileSizeErr: make(map[string]error),
	}
}

func (f *fakeFTPConn) RetrFrom(path string, offset uint64) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.retrOffsets = append(f.retrOffsets, offset)
	if f.retrErr != nil {
		return nil, f.retrErr
	}
	size := f.fileSizes[path]
	if offset >= uint64(size) {
		return io.NopCloser(bytes.NewReader(nil)), nil
	}
	payload := bytes.Repeat([]byte("x"), int(size-int64(offset)))
	return io.NopCloser(bytes.NewReader(payload)), nil
}

func (f *fakeFTPConn) Stor(path string, r io.Reader) error {
	return f.StorFrom(path, r, 0)
}

func (f *fakeFTPConn) StorFrom(path string, r io.Reader, offset uint64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.storOffsets = append(f.storOffsets, offset)
	if f.storErr != nil {
		return f.storErr
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if offset == 0 {
		f.storData = data
	} else {
		need := int(offset) + len(data)
		if need > len(f.storData) {
			padded := make([]byte, need)
			copy(padded, f.storData)
			f.storData = padded
		}
		copy(f.storData[offset:], data)
	}
	f.fileSizes[path] = int64(len(f.storData))
	return nil
}

func (f *fakeFTPConn) FileSize(path string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.fileSizeErr[path]; ok {
		return 0, err
	}
	size, ok := f.fileSizes[path]
	if !ok {
		return 0, &textproto.Error{Code: ftplib.StatusFileUnavailable, Msg: "not found"}
	}
	return size, nil
}

func (f *fakeFTPConn) List(string) ([]*ftplib.Entry, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listEntries, nil
}

func (f *fakeFTPConn) Walk(string) dirWalker {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.walk == nil {
		return &fakeWalker{}
	}
	return f.walk
}

func (f *fakeFTPConn) Quit() error {
	if f.quitErr != nil {
		return f.quitErr
	}
	return nil
}

func testClient(fc ftpConn, buf int) *ftpClient {
	if buf <= 0 {
		buf = defaultTransferBuffer
	}
	return &ftpClient{conn: fc, transferBuf: buf}
}
