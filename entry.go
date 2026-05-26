package ftp

import ftplib "github.com/jlaffaye/ftp"

// EntryType 远程条目类型。
type EntryType int

const (
	EntryTypeFile EntryType = iota
	EntryTypeFolder
	EntryTypeLink
)

// Entry 目录项元数据（List / Walk 使用）。
type Entry struct {
	Name string
	Size uint64
	Type EntryType
}

func entryFromFTP(e *ftplib.Entry) Entry {
	if e == nil {
		return Entry{}
	}
	out := Entry{Name: e.Name, Size: e.Size}
	switch e.Type {
	case ftplib.EntryTypeFolder:
		out.Type = EntryTypeFolder
	case ftplib.EntryTypeLink:
		out.Type = EntryTypeLink
	default:
		out.Type = EntryTypeFile
	}
	return out
}
