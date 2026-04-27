package script

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type memoryLogger struct {
	infos []string
	warns []string
	errs  []string
}

func (m *memoryLogger) Info(message string)  { m.infos = append(m.infos, message) }
func (m *memoryLogger) Warn(message string)  { m.warns = append(m.warns, message) }
func (m *memoryLogger) Error(message string) { m.errs = append(m.errs, message) }

func newTestWatcher(t *testing.T, root string) *moduleWatcher {
	t.Helper()

	logger := &memoryLogger{}
	return &moduleWatcher{
		rootDir:          root,
		interval:         "1m",
		logger:           logger,
		readFile:         os.ReadFile,
		walkDir:          filepath.WalkDir,
		now:              func() time.Time { return time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC) },
		baseline:         make(map[string]fileSnapshot),
		lastReadFailures: make(map[string]string),
	}
}

func TestModuleWatcherDetectsFileModification(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "nginx.yml")
	if err := os.WriteFile(file, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	assertContains(t, logger.infos, "change_type=modified")
	assertContains(t, logger.infos, "file=nginx.yml")
}

func TestModuleWatcherDetectsFileCreated(t *testing.T) {
	root := t.TempDir()
	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "redis.yml"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	assertContains(t, logger.infos, "change_type=created")
	assertContains(t, logger.infos, "file=redis.yml")
}

func TestModuleWatcherDetectsFileDeleted(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "mysql.yml")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	assertContains(t, logger.infos, "change_type=deleted")
	assertContains(t, logger.infos, "file=mysql.yml")
}

func TestModuleWatcherHandlesEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	if len(logger.warns) != 0 {
		t.Fatalf("expected no warnings, got %v", logger.warns)
	}
	if countChangeLogs(logger.infos) != 0 {
		t.Fatalf("expected no change logs, got %v", logger.infos)
	}
}

func TestModuleWatcherHandlesReadError(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "bad.yml")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)
	watcher.readFile = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "bad.yml") {
			return nil, errors.New("boom")
		}
		return os.ReadFile(path)
	}

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}
	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	if len(logger.warns) != 1 {
		t.Fatalf("expected one warning log, got %v", logger.warns)
	}
	assertContains(t, logger.warns, "change_type=read_error")
}

func TestModuleWatcherPreservesFileOnReadFailure(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "bad.yml")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	watcher := newTestWatcher(t, root)
	logger := watcher.logger.(*memoryLogger)

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	watcher.readFile = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "bad.yml") {
			return nil, errors.New("boom")
		}
		return os.ReadFile(path)
	}

	if err := watcher.Run(); err != nil {
		t.Fatal(err)
	}

	for _, info := range logger.infos {
		if strings.Contains(info, "change_type=deleted") {
			t.Fatalf("unexpected deleted log: %s", info)
		}
	}
}

func countChangeLogs(messages []string) int {
	total := 0
	for _, message := range messages {
		if strings.Contains(message, "change_type=") && !strings.Contains(message, "baseline initialized") {
			total++
		}
	}
	return total
}

func assertContains(t *testing.T, messages []string, expected string) {
	t.Helper()
	for _, message := range messages {
		if strings.Contains(message, expected) {
			return
		}
	}
	t.Fatalf("expected %q in %v", expected, messages)
}
