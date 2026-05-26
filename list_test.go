package ftp

import (
	"context"
	"errors"
	"io"
	"testing"

	ftplib "github.com/jlaffaye/ftp"
)

func TestList_success(t *testing.T) {
	fc := newFakeFTPConn()
	fc.listEntries = []*ftplib.Entry{{Name: "a"}, {Name: "b"}}
	c := testClient(fc, 1024)
	names, err := c.List(context.Background(), "/")
	if err != nil || len(names) != 2 {
		t.Fatalf("names=%v err=%v", names, err)
	}
}

func TestList_errors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var nilClient *ftpClient
	if _, err := nilClient.List(ctx, "/"); err == nil {
		t.Fatal("nil client")
	}
	c := testClient(newFakeFTPConn(), 1024)
	c.conn = nil
	if _, err := c.List(context.Background(), "/"); err == nil {
		t.Fatal("nil conn")
	}
	fc := newFakeFTPConn()
	fc.listErr = errors.New("list fail")
	c = testClient(fc, 1024)
	if _, err := c.List(context.Background(), "/"); err == nil {
		t.Fatal("list err")
	}
	if _, err := c.List(ctx, "/"); err == nil {
		t.Fatal("ctx")
	}
}

func TestListEach_successAndStop(t *testing.T) {
	fc := newFakeFTPConn()
	fc.listEntries = []*ftplib.Entry{
		{Name: "a", Type: ftplib.EntryTypeFile},
		{Name: "b", Type: ftplib.EntryTypeFolder},
	}
	c := testClient(fc, 1024)
	var n int
	if err := c.ListEach(context.Background(), "/", func(e Entry) error {
		n++
		if e.Name == "a" {
			return ErrStopIteration
		}
		return nil
	}); err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestListEach_errors(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	if err := c.ListEach(context.Background(), "/", nil); err == nil {
		t.Fatal("nil fn")
	}
	var nilClient *ftpClient
	if err := nilClient.ListEach(context.Background(), "/", func(Entry) error { return nil }); err == nil {
		t.Fatal("nil client")
	}
	fc := newFakeFTPConn()
	fc.listErr = errors.New("x")
	c = testClient(fc, 1024)
	if err := c.ListEach(context.Background(), "/", func(Entry) error { return errors.New("cb") }); err == nil {
		t.Fatal("list")
	}
	fc = newFakeFTPConn()
	fc.listEntries = []*ftplib.Entry{{Name: "a"}}
	c = testClient(fc, 1024)
	if err := c.ListEach(context.Background(), "/", func(Entry) error { return errors.New("cb") }); err == nil {
		t.Fatal("cb err")
	}
}

func TestWalkEach_success(t *testing.T) {
	fc := newFakeFTPConn()
	fc.walk = &fakeWalker{
		paths:   []string{"/a", "/b"},
		entries: []*ftplib.Entry{{Name: "a", Type: ftplib.EntryTypeFile}, {Name: "b", Type: ftplib.EntryTypeFolder}},
		idx:     -1,
	}
	c := testClient(fc, 1024)
	var names []string
	if err := c.WalkEach(context.Background(), "/", func(e Entry) error {
		names = append(names, e.Name)
		if e.Name == "/a" {
			return ErrStopIteration
		}
		return nil
	}); err != nil || len(names) != 1 {
		t.Fatalf("names=%v err=%v", names, err)
	}
}

func TestWalkEach_walkError(t *testing.T) {
	fc := newFakeFTPConn()
	fc.walk = &fakeWalker{
		paths:   []string{"/a"},
		entries: []*ftplib.Entry{{Name: "a"}},
		idx:     -1,
		errAt:   0,
		walkErr: errors.New("walk fail"),
	}
	c := testClient(fc, 1024)
	if err := c.WalkEach(context.Background(), "/", func(Entry) error { return nil }); err == nil {
		t.Fatal("expected walk error")
	}
}

func TestWalkEach_errors(t *testing.T) {
	c := testClient(newFakeFTPConn(), 1024)
	if err := c.WalkEach(context.Background(), "/", nil); err == nil {
		t.Fatal("nil fn")
	}
	var nilClient *ftpClient
	if err := nilClient.WalkEach(context.Background(), "/", func(Entry) error { return nil }); err == nil {
		t.Fatal("nil client")
	}
}

func TestListIter_success(t *testing.T) {
	fc := newFakeFTPConn()
	fc.listEntries = []*ftplib.Entry{{Name: "f", Type: ftplib.EntryTypeFile}}
	c := testClient(fc, 1024)
	it, err := c.ListIter(context.Background(), "/")
	if err != nil {
		t.Fatal(err)
	}
	e, err := it.Next()
	if err != nil || e.Name != "f" {
		t.Fatalf("e=%+v err=%v", e, err)
	}
	_, err = it.Next()
	if err != io.EOF {
		t.Fatal(err)
	}
}

func TestListIter_errors(t *testing.T) {
	var nilIter *ListIter
	if _, err := nilIter.Next(); err != io.EOF {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	it := &ListIter{ctx: ctx, entries: []*Entry{{Name: "a"}}}
	if _, err := it.Next(); err == nil {
		t.Fatal("ctx")
	}
	var nilClient *ftpClient
	if _, err := nilClient.ListIter(context.Background(), "/"); err == nil {
		t.Fatal("nil client")
	}
	fc := newFakeFTPConn()
	fc.listErr = errors.New("list")
	c := testClient(fc, 1024)
	if _, err := c.ListIter(context.Background(), "/"); err == nil {
		t.Fatal("list")
	}
}
