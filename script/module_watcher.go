package script

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	filebeatConfig "github.com/fufuok/beats-http-output/config"
	infraLog "github.com/fufuok/beats-http-output/infra"
)

const ModuleWatcherScriptName = "module_watcher"

type logWriter interface {
	Info(message string)
	Warn(message string)
	Error(message string)
}

type fileSnapshot struct {
	Hash string
}

type moduleWatcher struct {
	rootDir          string
	interval         string
	logger           logWriter
	readFile         func(string) ([]byte, error)
	walkDir          func(string, fs.WalkDirFunc) error
	now              func() time.Time
	mu               sync.Mutex
	initialized      bool
	baseline         map[string]fileSnapshot
	lastReadFailures map[string]string
}

func init() {
	Register(ModuleWatcherScriptName, func(cfg *filebeatConfig.FilebeatConfig) (Script, error) {
		if cfg == nil || !cfg.Scripts.ModuleWatcher.Enabled {
			return nil, nil
		}
		return NewModuleWatcher(cfg.Scripts.ModuleWatcher)
	})
}

func NewModuleWatcher(cfg filebeatConfig.ModuleWatcherConfig) (Script, error) {
	if cfg.Directory == "" {
		cfg.Directory = "./modules.d"
	}
	if cfg.Interval == "" {
		cfg.Interval = "1m"
	}

	return &moduleWatcher{
		rootDir:          cfg.Directory,
		interval:         cfg.Interval,
		logger:           infraLog.GlobalLog,
		readFile:         os.ReadFile,
		walkDir:          filepath.WalkDir,
		now:              time.Now,
		baseline:         make(map[string]fileSnapshot),
		lastReadFailures: make(map[string]string),
	}, nil
}

func (m *moduleWatcher) Name() string {
	return ModuleWatcherScriptName
}

func (m *moduleWatcher) Interval() string {
	return m.interval
}

func (m *moduleWatcher) Run() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	current, err := m.scan()
	if err != nil {
		return err
	}

	if !m.initialized {
		m.baseline = current
		m.initialized = true
		m.logger.Info(fmt.Sprintf("script=%s baseline initialized directory=%s file_count=%d", m.Name(), m.rootDir, len(current)))
		return nil
	}

	for _, message := range diffSnapshots(m.baseline, current, m.now) {
		m.logger.Info(message)
	}

	m.baseline = current
	return nil
}

func (m *moduleWatcher) scan() (map[string]fileSnapshot, error) {
	files := make(map[string]fileSnapshot)

	err := m.walkDir(m.rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			m.logger.Warn(fmt.Sprintf("script=%s file=%s change_type=read_error occurred_at=%s error=%v", m.Name(), path, m.now().Format(time.RFC3339), walkErr))
			return nil
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(m.rootDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		content, err := m.readFile(path)
		if err != nil {
			m.handleReadFailure(relPath, err)
			if previous, ok := m.baseline[relPath]; ok {
				files[relPath] = previous
			}
			return nil
		}

		delete(m.lastReadFailures, relPath)
		files[relPath] = fileSnapshot{Hash: hashContent(content)}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (m *moduleWatcher) handleReadFailure(file string, err error) {
	message := err.Error()
	if last, ok := m.lastReadFailures[file]; ok && last == message {
		return
	}

	m.lastReadFailures[file] = message
	m.logger.Warn(fmt.Sprintf("script=%s file=%s change_type=read_error occurred_at=%s error=%v", m.Name(), file, m.now().Format(time.RFC3339), err))
}

func diffSnapshots(previous, current map[string]fileSnapshot, now func() time.Time) []string {
	var messages []string
	timestamp := now().Format(time.RFC3339)

	for file, snapshot := range current {
		previousSnapshot, exists := previous[file]
		if !exists {
			messages = append(messages, fmt.Sprintf("script=%s file=%s change_type=created occurred_at=%s", ModuleWatcherScriptName, file, timestamp))
			continue
		}
		if previousSnapshot.Hash != snapshot.Hash {
			messages = append(messages, fmt.Sprintf("script=%s file=%s change_type=modified occurred_at=%s", ModuleWatcherScriptName, file, timestamp))
		}
	}

	for file := range previous {
		if _, exists := current[file]; !exists {
			messages = append(messages, fmt.Sprintf("script=%s file=%s change_type=deleted occurred_at=%s", ModuleWatcherScriptName, file, timestamp))
		}
	}

	sort.Strings(messages)
	return messages
}

func hashContent(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
