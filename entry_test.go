package ftp

import (
	"testing"

	ftplib "github.com/jlaffaye/ftp"
)

func TestEntryFromFTP_nil(t *testing.T) {
	if e := entryFromFTP(nil); e.Name != "" || e.Type != EntryTypeFile {
		t.Fatalf("%+v", e)
	}
}

func TestEntryFromFTP_types(t *testing.T) {
	cases := []struct {
		in   ftplib.EntryType
		want EntryType
	}{
		{ftplib.EntryTypeFolder, EntryTypeFolder},
		{ftplib.EntryTypeLink, EntryTypeLink},
		{ftplib.EntryTypeFile, EntryTypeFile},
		{ftplib.EntryType(99), EntryTypeFile},
	}
	for _, tc := range cases {
		e := entryFromFTP(&ftplib.Entry{Name: "x", Size: 7, Type: tc.in})
		if e.Type != tc.want || e.Size != 7 {
			t.Fatalf("in=%v got=%+v", tc.in, e)
		}
	}
}
