package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// ErrStopIteration 在 ListEach / WalkEach 回调中返回可提前结束遍历（不视为错误）。
var ErrStopIteration = errors.New("ftp: stop iteration")

// ListEach 逐条处理目录项，不向调用方返回巨型 []string。
// 注意：标准 FTP LIST 一次拉全目录，库内仍会短暂持有服务端返回列表；超大规模目录请分子目录或 Walk 拆分。
func (c *ftpClient) ListEach(ctx context.Context, remoteDir string, fn func(Entry) error) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	if fn == nil {
		return fmt.Errorf("ftp: list callback is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	entries, err := c.conn.List(remoteDir)
	if err != nil {
		return fmt.Errorf("ftp list: %w", err)
	}
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(entryFromFTP(e)); err != nil {
			if errors.Is(err, ErrStopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

// WalkEach 递归遍历目录树，按 Walk 顺序逐条回调（适合大量文件分批处理，避免一次性 List 根目录）。
func (c *ftpClient) WalkEach(ctx context.Context, root string, fn func(Entry) error) error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("ftp: not connected")
	}
	if fn == nil {
		return fmt.Errorf("ftp: walk callback is nil")
	}
	w := c.conn.Walk(root)
	for w.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := w.Err(); err != nil {
			return fmt.Errorf("ftp walk: %w", err)
		}
		ent := entryFromFTP(w.Stat())
		ent.Name = w.Path()
		if err := fn(ent); err != nil {
			if errors.Is(err, ErrStopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

// ListIter 拉取目录后按 Next 逐条读取，调用方不必持有完整切片。
type ListIter struct {
	ctx     context.Context
	entries []*Entry
	idx     int
}

// Next 返回下一条；io.EOF 表示结束。
func (it *ListIter) Next() (Entry, error) {
	if it == nil {
		return Entry{}, io.EOF
	}
	if err := it.ctx.Err(); err != nil {
		return Entry{}, err
	}
	if it.idx >= len(it.entries) {
		return Entry{}, io.EOF
	}
	e := *it.entries[it.idx]
	it.idx++
	return e, nil
}

// ListIter 创建目录迭代器（内部分配条目切片，大目录优先 ListEach / WalkEach）。
func (c *ftpClient) ListIter(ctx context.Context, remoteDir string) (*ListIter, error) {
	if c == nil || c.conn == nil {
		return nil, fmt.Errorf("ftp: not connected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, err := c.conn.List(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("ftp list: %w", err)
	}
	entries := make([]*Entry, len(raw))
	for i, e := range raw {
		ent := entryFromFTP(e)
		entries[i] = &ent
	}
	return &ListIter{ctx: ctx, entries: entries}, nil
}
