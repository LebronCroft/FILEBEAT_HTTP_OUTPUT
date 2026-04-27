package script

import (
	filebeatConfig "github.com/fufuok/beats-http-output/config"
	"sync"
)

type Script interface {
	Name() string
	Interval() string
	Run() error
}

type Factory func(cfg *filebeatConfig.FilebeatConfig) (Script, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

func Register(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

func Registered() map[string]Factory {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[string]Factory, len(registry))
	for name, factory := range registry {
		result[name] = factory
	}
	return result
}
