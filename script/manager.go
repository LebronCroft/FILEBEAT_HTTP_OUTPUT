package script

import (
	"fmt"
	"sync"
	"time"

	filebeatConfig "github.com/fufuok/beats-http-output/config"
	infraLog "github.com/fufuok/beats-http-output/infra"
)

type Manager struct {
	scripts []Script
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func NewManager(cfg *filebeatConfig.FilebeatConfig) (*Manager, error) {
	manager := &Manager{
		stopCh: make(chan struct{}),
	}

	for name, factory := range Registered() {
		instance, err := factory(cfg)
		if err != nil {
			return nil, fmt.Errorf("create script %s failed: %w", name, err)
		}
		if instance != nil {
			manager.scripts = append(manager.scripts, instance)
		}
	}

	return manager, nil
}

func (m *Manager) Start() error {
	for _, script := range m.scripts {
		interval, err := time.ParseDuration(script.Interval())
		if err != nil {
			return fmt.Errorf("invalid interval for script %s: %w", script.Name(), err)
		}

		m.wg.Add(1)
		go m.runScript(script, interval)
	}

	return nil
}

func (m *Manager) Stop() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
	m.wg.Wait()
}

func (m *Manager) runScript(script Script, interval time.Duration) {
	defer m.wg.Done()

	if err := script.Run(); err != nil {
		infraLog.GlobalLog.Error(fmt.Sprintf("script %s initial run failed: %v", script.Name(), err))
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := script.Run(); err != nil {
				infraLog.GlobalLog.Error(fmt.Sprintf("script %s run failed: %v", script.Name(), err))
			}
		case <-m.stopCh:
			return
		}
	}
}
