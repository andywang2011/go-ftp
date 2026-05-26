package ftp

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestCopyWithContext_success(t *testing.T) {
	src := strings.NewReader("hello-large-chunk-test")
	var dst bytes.Buffer
	buf := make([]byte, 8)
	n, err := copyWithContext(context.Background(), &dst, src, buf)
	if err != nil || n == 0 || dst.Len() == 0 {
		t.Fatalf("n=%d err=%v dst=%q", n, err, dst.String())
	}
}

func TestCopyWithContext_cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := strings.NewReader(strings.Repeat("x", 1024))
	var dst bytes.Buffer
	_, err := copyWithContext(ctx, &dst, src, make([]byte, 32))
	if err == nil {
		t.Fatal("expected cancel error")
	}
}

func TestContextReader_cancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)
	cr := &contextReader{ctx: ctx, r: strings.NewReader("ab")}
	buf := make([]byte, 4)
	_, err := cr.Read(buf)
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestApplyTransferOpts_default(t *testing.T) {
	o := applyTransferOpts(nil, 1024)
	if o.bufferSize != 1024 {
		t.Fatal(o.bufferSize)
	}
}

func TestWithBufferSize(t *testing.T) {
	o := applyTransferOpts([]TransferOption{WithBufferSize(512 * 1024)}, 0)
	if o.bufferSize != 512*1024 {
		t.Fatal(o.bufferSize)
	}
}

func TestListIter_EOF(t *testing.T) {
	it := &ListIter{ctx: context.Background(), entries: []*Entry{}}
	_, err := it.Next()
	if err != io.EOF {
		t.Fatal(err)
	}
}
