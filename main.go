// Command orgo serves a browsable, live-reloading view of an org-roam
// directory: `orgo [flags] [dir]`. See docs/DESIGN.md for the full design.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/naok-000/orgo/internal/roam"
	"github.com/naok-000/orgo/internal/server"
	"github.com/naok-000/orgo/internal/watch"
)

// version is the orgo release version. It's a var so it can be overridden at
// build time, e.g. `go build -ldflags "-X main.version=1.2.3"`.
var version = "dev"

const (
	defaultPort = 35911
	defaultAddr = "127.0.0.1"
	debounce    = 300 * time.Millisecond
)

// config holds the parsed CLI flags/arguments.
type config struct {
	port      int
	addr      string
	noBrowser bool
	version   bool
	dir       string
}

// parseArgs parses args (as in os.Args[1:]) into a config. It is separated
// from run so the CLI surface can be unit tested without touching the
// network or filesystem.
func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("orgo", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: orgo [flags] [dir]")
		fs.PrintDefaults()
	}

	var cfg config
	fs.IntVar(&cfg.port, "port", defaultPort, "port to listen on")
	fs.IntVar(&cfg.port, "p", defaultPort, "port to listen on (shorthand)")
	fs.StringVar(&cfg.addr, "addr", defaultAddr, "address to bind")
	fs.BoolVar(&cfg.noBrowser, "no-browser", false, "don't auto-open the browser on start")
	fs.BoolVar(&cfg.version, "version", false, "print the version and exit")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}

	cfg.dir = "."
	if fs.NArg() > 0 {
		cfg.dir = fs.Arg(0)
	}
	if fs.NArg() > 1 {
		return config{}, fmt.Errorf("unexpected extra arguments: %v", fs.Args()[1:])
	}
	return cfg, nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "orgo:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}

	if cfg.version {
		fmt.Fprintln(stdout, "orgo "+version)
		return nil
	}

	absDir, err := filepath.Abs(cfg.dir)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", cfg.dir, err)
	}
	if info, err := os.Stat(absDir); err != nil {
		return fmt.Errorf("cannot access %s: %w", absDir, err)
	} else if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", absDir)
	}

	// Bind the port before the (potentially long) initial scan so clients —
	// including orgo.el, which polls the port for readiness — can connect
	// immediately; their requests queue in the accept backlog until Serve
	// starts below.
	ln, err := net.Listen("tcp", net.JoinHostPort(cfg.addr, strconv.Itoa(cfg.port)))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer ln.Close()

	url := fmt.Sprintf("http://%s/", ln.Addr().String())
	log.Printf("orgo: listening at %s", url)

	idx, err := roam.Scan(absDir)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", absDir, err)
	}
	log.Printf("orgo: indexed %d notes from %s", idx.NoteCount(), absDir)

	srv := server.New(idx, version)
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
		// DNS-rebinding protection: only expected Host headers are served
		// when bound to loopback (the default).
		srv.RestrictHost(cfg.addr, tcpAddr.Port)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	w, err := watch.New(absDir, debounce, func() {
		newIdx, err := roam.Scan(absDir)
		if err != nil {
			log.Printf("orgo: re-index failed: %v", err)
			return
		}
		srv.SetIndex(newIdx)
		log.Printf("orgo: re-indexed %d notes", newIdx.NoteCount())
	})
	if err != nil {
		return fmt.Errorf("starting watcher: %w", err)
	}
	defer w.Close()
	if err := w.Start(ctx); err != nil {
		return fmt.Errorf("watcher: %w", err)
	}

	log.Printf("orgo: serving %s at %s", absDir, url)

	httpServer := &http.Server{Handler: srv}
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(ln) }()

	if !cfg.noBrowser {
		openBrowser(url)
	}

	select {
	case <-ctx.Done():
		log.Println("orgo: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// openBrowser best-effort opens url in the user's default browser. It never
// fails the program: if the platform isn't supported or the command isn't
// found, it just logs and moves on.
func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		log.Printf("orgo: don't know how to open a browser on %s; open %s manually", runtime.GOOS, url)
		return
	}
	if err := exec.Command(cmd, url).Start(); err != nil {
		log.Printf("orgo: failed to open browser: %v", err)
	}
}
