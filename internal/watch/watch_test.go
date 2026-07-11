package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testDebounce = 30 * time.Millisecond

func waitForChange(t *testing.T, ch <-chan struct{}, timeout time.Duration) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(timeout):
		t.Fatal("timed out waiting for onChange callback")
	}
}

func assertNoChange(t *testing.T, ch <-chan struct{}, wait time.Duration) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("unexpected onChange callback")
	case <-time.After(wait):
	}
}

func newTestWatcher(t *testing.T, root string) (*Watcher, <-chan struct{}) {
	t.Helper()
	changed := make(chan struct{}, 16)
	w, err := New(root, testDebounce, func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		w.Close()
	})
	return w, changed
}

func TestWatcherFiresOnFileCreate(t *testing.T) {
	root := t.TempDir()
	_, changed := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, "new.org"), []byte("#+title: New\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForChange(t, changed, 2*time.Second)
}

func TestWatcherFiresOnFileModify(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "existing.org")
	if err := os.WriteFile(target, []byte("#+title: Existing\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, changed := newTestWatcher(t, root)

	if err := os.WriteFile(target, []byte("#+title: Existing\n\nUpdated body.\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForChange(t, changed, 2*time.Second)
}

func TestWatcherIgnoresNonOrgFiles(t *testing.T) {
	root := t.TempDir()
	_, changed := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	assertNoChange(t, changed, 200*time.Millisecond)
}

func TestWatcherIgnoresDotfiles(t *testing.T) {
	root := t.TempDir()
	_, changed := newTestWatcher(t, root)

	if err := os.WriteFile(filepath.Join(root, ".hidden.org"), []byte("#+title: Hidden\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	assertNoChange(t, changed, 200*time.Millisecond)
}

func TestWatcherDebouncesBurstsIntoOneCallback(t *testing.T) {
	root := t.TempDir()
	_, changed := newTestWatcher(t, root)

	target := filepath.Join(root, "burst.org")
	for i := 0; i < 5; i++ {
		if err := os.WriteFile(target, []byte("burst"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		time.Sleep(testDebounce / 3)
	}

	waitForChange(t, changed, 2*time.Second)
	// Give it a moment: if each write triggered its own callback we'd see a
	// second one queued up almost immediately.
	assertNoChange(t, changed, testDebounce*2)
}

func TestWatcherDetectsFileInNewSubdirectory(t *testing.T) {
	root := t.TempDir()
	_, changed := newTestWatcher(t, root)

	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	// Give the watcher a moment to notice the new directory and add a watch
	// for it before we write into it.
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(sub, "nested.org"), []byte("#+title: Nested\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	waitForChange(t, changed, 2*time.Second)
}
