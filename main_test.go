package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseArgsDefaults(t *testing.T) {
	cfg, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if cfg.port != defaultPort {
		t.Errorf("port = %d, want %d", cfg.port, defaultPort)
	}
	if cfg.addr != defaultAddr {
		t.Errorf("addr = %q, want %q", cfg.addr, defaultAddr)
	}
	if cfg.noBrowser {
		t.Error("noBrowser should default to false")
	}
	if cfg.version {
		t.Error("version should default to false")
	}
	if cfg.dir != "." {
		t.Errorf("dir = %q, want %q (default)", cfg.dir, ".")
	}
}

func TestParseArgsPortShortAndLongAliasesAgree(t *testing.T) {
	shortForm, err := parseArgs([]string{"-p", "8080"})
	if err != nil {
		t.Fatalf("parseArgs(-p): %v", err)
	}
	longForm, err := parseArgs([]string{"--port", "8080"})
	if err != nil {
		t.Fatalf("parseArgs(--port): %v", err)
	}
	if shortForm.port != 8080 || longForm.port != 8080 {
		t.Errorf("port short=%d long=%d, want both 8080", shortForm.port, longForm.port)
	}
}

func TestParseArgsDirPositional(t *testing.T) {
	cfg, err := parseArgs([]string{"--no-browser", "/some/notes"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if cfg.dir != "/some/notes" {
		t.Errorf("dir = %q, want /some/notes", cfg.dir)
	}
	if !cfg.noBrowser {
		t.Error("noBrowser should be true")
	}
}

func TestParseArgsAddrAndVersion(t *testing.T) {
	cfg, err := parseArgs([]string{"--addr", "0.0.0.0", "--version"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if cfg.addr != "0.0.0.0" {
		t.Errorf("addr = %q, want 0.0.0.0", cfg.addr)
	}
	if !cfg.version {
		t.Error("version should be true")
	}
}

func TestParseArgsRejectsExtraPositionalArgs(t *testing.T) {
	if _, err := parseArgs([]string{"dir1", "dir2"}); err == nil {
		t.Error("expected an error for more than one positional argument")
	}
}

func TestParseArgsRejectsUnknownFlag(t *testing.T) {
	if _, err := parseArgs([]string{"--nope"}); err == nil {
		t.Error("expected an error for an unknown flag")
	}
}

func TestRunVersionFlagPrintsAndExitsCleanly(t *testing.T) {
	var buf bytes.Buffer
	if err := run([]string{"--version"}, &buf); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "orgo") {
		t.Errorf("output = %q, want it to mention orgo", buf.String())
	}
}

func TestRunRejectsNonDirectoryTarget(t *testing.T) {
	var buf bytes.Buffer
	err := run([]string{"main_test.go"}, &buf) // this file is not a directory
	if err == nil {
		t.Fatal("expected an error targeting a non-directory")
	}
}

func TestRunRejectsMissingDirectory(t *testing.T) {
	var buf bytes.Buffer
	err := run([]string{"/definitely/does/not/exist/orgo-test"}, &buf)
	if err == nil {
		t.Fatal("expected an error for a missing directory")
	}
}
