package ftp_test

import (
	"fmt"

	"github.com/andywang2011/go-ftp"
)

func ExampleConfigDefault() {
	cfg := ftp.ConfigDefault()
	fmt.Println(cfg.Port)
	// Output: 21
}

func ExampleConfig_validate() {
	cfg := ftp.ConfigDefault()
	err := cfg.Validate()
	fmt.Println(err != nil)
	// Output: true
}

func ExampleEntryType() {
	fmt.Println(int(ftp.EntryTypeFile))
	// Output: 0
}
