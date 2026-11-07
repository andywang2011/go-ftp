package ftp

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Lister lists and walks remote directories.
type Lister interface {
	// List returns entry names in remoteDir.
	List(ctx context.Context, remoteDir string) ([]string, error)
	// ListEach invokes fn for each entry without collecting a full slice.
	ListEach(ctx context.Context, remoteDir string, fn func(Entry) error) error
	// ListIter returns a ListIter (same name as type; v0.2 may rename to OpenDirIter).
	ListIter(ctx context.Context, remoteDir string) (*ListIter, error)
	// WalkEach invokes fn for each entry under root recursively.
	WalkEach(ctx context.Context, root string, fn func(Entry) error) error
}

var _ Lister = (*ftpClient)(nil)

func (c *ftpClient) List(ctx context.Context, remoteDir string) ([]string, error) {
	if c == nil || c.conn == nil {
		return nil, ErrNotConnected
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := c.conn.List(remoteDir)
	if err != nil {
		return nil, fmt.Errorf("ftp list: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names, nil
}

// ListEach invokes fn per entry without returning a large []string to the caller.
// The server still returns the full LIST in one round trip; use WalkEach or split dirs at huge scale.
func (c *ftpClient) ListEach(ctx context.Context, remoteDir string, fn func(Entry) error) error {
	if c == nil || c.conn == nil {
		return ErrNotConnected
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

// WalkEach walks a directory tree in Walk order; Entry.Name is the full path from the walker.
func (c *ftpClient) WalkEach(ctx context.Context, root string, fn func(Entry) error) error {
	if c == nil || c.conn == nil {
		return ErrNotConnected
	}
	if fn == nil {
		return fmt.Errorf("ftp: walk callback is nil")
	}
	walker := c.conn.Walk(root)
	if walker == nil {
		return fmt.Errorf("ftp: walk unavailable")
	}
	for walker.Next() {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := walker.Err(); err != nil {
			return fmt.Errorf("ftp walk: %w", err)
		}
		ent := entryFromFTP(walker.Stat())
		ent.Name = walker.Path()
		if err := fn(ent); err != nil {
			if errors.Is(err, ErrStopIteration) {
				return nil
			}
			return err
		}
	}
	return nil
}

// ListIter iterates directory entries after one LIST (holds a slice internally; prefer ListEach when possible).
type ListIter struct {
	ctx     context.Context
	entries []*Entry
	idx     int
}

// Next returns the next entry or io.EOF when finished.
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

// ListIter fetches remoteDir and returns an iterator over entries.
func (c *ftpClient) ListIter(ctx context.Context, remoteDir string) (*ListIter, error) {
	if c == nil || c.conn == nil {
		return nil, ErrNotConnected
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
