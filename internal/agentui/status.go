package agentui

import (
	"sync"
	"time"
)

type HeartbeatState string

const (
	HeartbeatUnknown HeartbeatState = "unknown"
	HeartbeatOK      HeartbeatState = "ok"
	HeartbeatFail    HeartbeatState = "fail"
)

type HeartbeatStatus struct {
	State     HeartbeatState
	Target    string
	Message   string
	Timestamp time.Time
}

type StatusManager struct {
	mu          sync.RWMutex
	status      HeartbeatStatus
	subscribers []chan struct{}
}

func NewStatusManager() *StatusManager {
	return &StatusManager{
		status: HeartbeatStatus{
			State:     HeartbeatUnknown,
			Timestamp: time.Now(),
		},
		subscribers: make([]chan struct{}, 0),
	}
}

func (sm *StatusManager) Get() HeartbeatStatus {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.status
}

func (sm *StatusManager) Set(status HeartbeatStatus) {
	sm.mu.Lock()
	sm.status = status
	sm.mu.Unlock()
	sm.notifySubscribers()
}

func (sm *StatusManager) Subscribe() <-chan struct{} {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	ch := make(chan struct{}, 10)
	sm.subscribers = append(sm.subscribers, ch)
	return ch
}

func (sm *StatusManager) Unsubscribe(ch <-chan struct{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, sub := range sm.subscribers {
		if sub == ch {
			close(sub)
			sm.subscribers = append(sm.subscribers[:i], sm.subscribers[i+1:]...)
			return
		}
	}
}

func (sm *StatusManager) notifySubscribers() {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, ch := range sm.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
