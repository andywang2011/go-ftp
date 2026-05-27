package ftp

import "testing"

func TestConfig_Validate(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", User: "u", Password: "p"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != defaultPort {
		t.Fatal(cfg.Port)
	}
}

func TestConfig_Validate_missingUser(t *testing.T) {
	cfg := Config{Host: "h", Password: "p"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_Validate_missingPassword(t *testing.T) {
	cfg := Config{Host: "h", User: "u"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_Addr(t *testing.T) {
	cfg := Config{Host: "h", Port: 2121, User: "u", Password: "p"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Addr() != "h:2121" {
		t.Fatal(cfg.Addr())
	}
}

func TestNewClient_invalidHost(t *testing.T) {
	_, err := NewClient(Config{Host: "127.0.0.1", Port: 59999, User: "u", Password: "p"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewClient_missingCredentials(t *testing.T) {
	_, err := NewClient(Config{Host: "127.0.0.1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_Validate_missingHost(t *testing.T) {
	cfg := Config{User: "u", Password: "p"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

func TestConfig_transferBufferSize(t *testing.T) {
	if n := (Config{TransferBufferSize: 8192}).transferBufferSize(); n != 8192 {
		t.Fatal(n)
	}
	if n := (Config{}).transferBufferSize(); n != defaultTransferBuffer {
		t.Fatal(n)
	}
}

func TestConfigDefault(t *testing.T) {
	c := ConfigDefault()
	if c.Port != defaultPort || c.Timeout == 0 || c.TransferBufferSize != defaultTransferBuffer {
		t.Fatalf("%+v", c)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected validate error without host")
	}
}
