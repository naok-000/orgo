// Package watch watches a directory tree for filesystem changes and, after a
// debounce period of quiet, invokes a callback. It knows nothing about org
// files or indexing specifically; the caller decides what "changed" means
// beyond "not a dotfile" and what to do about it (re-index, broadcast SSE,
// ...).
package watch

import (
	"context"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches root recursively (including directories created after
// startup) and calls onChange after debounce has passed with no further
// filesystem activity.
type Watcher struct {
	root     string
	debounce time.Duration
	onChange func()
	fsw      *fsnotify.Watcher
}

// New creates a Watcher rooted at root. It does not start watching until
// Start is called.
func New(root string, debounce time.Duration, onChange func()) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{root: root, debounce: debounce, onChange: onChange, fsw: fsw}
	if err := w.addDirs(root); err != nil {
		fsw.Close()
		return nil, err
	}
	return w, nil
}

// Start begins watching in the background. It returns once the watch loop
// goroutine has been launched; the loop stops when ctx is done.
func (w *Watcher) Start(ctx context.Context) error {
	go w.loop(ctx)
	return nil
}

// Close releases the underlying filesystem watch.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}

func (w *Watcher) addDirs(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("watch: skipping %s: %v", path, err)
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if path != root && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}
		if err := w.fsw.Add(path); err != nil {
			log.Printf("watch: failed to watch %s: %v", path, err)
		}
		return nil
	})
}

func (w *Watcher) loop(ctx context.Context) {
	var timer *time.Timer
	var timerC <-chan time.Time

	resetTimer := func() {
		if timer == nil {
			timer = time.NewTimer(w.debounce)
		} else {
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(w.debounce)
		}
		timerC = timer.C
	}

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			name := filepath.Base(ev.Name)
			if strings.HasPrefix(name, ".") {
				continue
			}
			if ev.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
					if err := w.addDirs(ev.Name); err != nil {
						log.Printf("watch: %v", err)
					}
				}
			}
			if strings.EqualFold(filepath.Ext(name), ".org") {
				resetTimer()
			}

		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watch: error: %v", err)

		case <-timerC:
			timerC = nil
			w.onChange()
		}
	}
}
