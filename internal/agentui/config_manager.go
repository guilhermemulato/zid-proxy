package agentui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ConfigManager struct {
	mu          sync.RWMutex
	path        string
	cfg         Config
	subscribers []chan struct{}
}

func NewConfigManager(path string, initial Config) *ConfigManager {
	return &ConfigManager{
		path:        path,
		cfg:         initial,
		subscribers: make([]chan struct{}, 0),
	}
}

func (cm *ConfigManager) Path() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.path
}

func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg
}

func (cm *ConfigManager) Set(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	cm.mu.Lock()
	cm.cfg = cfg
	cm.mu.Unlock()

	cm.notifySubscribers()
	return nil
}

func (cm *ConfigManager) LoadFromDisk() error {
	cm.mu.RLock()
	path := cm.path
	cm.mu.RUnlock()

	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	return cm.Set(cfg)
}

func (cm *ConfigManager) SaveToDisk() error {
	cfg := cm.Get()

	if err := cfg.Validate(); err != nil {
		return err
	}

	cm.mu.RLock()
	path := cm.path
	cm.mu.RUnlock()

	if path == "" {
		return os.ErrInvalid
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (cm *ConfigManager) Subscribe() <-chan struct{} {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	ch := make(chan struct{}, 10)
	cm.subscribers = append(cm.subscribers, ch)
	return ch
}

func (cm *ConfigManager) Unsubscribe(ch <-chan struct{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, sub := range cm.subscribers {
		if sub == ch {
			close(sub)
			cm.subscribers = append(cm.subscribers[:i], cm.subscribers[i+1:]...)
			return
		}
	}
}

func (cm *ConfigManager) notifySubscribers() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, ch := range cm.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
