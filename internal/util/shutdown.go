package util

import (
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// Manager coordinates graceful shutdown handlers.
// Handlers are executed in reverse registration order (LIFO) so that
// dependencies are torn down after their consumers.
type Manager struct {
	mu       sync.Mutex
	handlers []func() error
	once     sync.Once
}

// NewManager creates a shutdown manager.
func NewManager() *Manager {
	return &Manager{}
}

// Register adds a cleanup handler. Handlers are called in reverse order.
func (m *Manager) Register(fn func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, fn)
}

// Listen starts a goroutine that waits for SIGINT/SIGTERM and then calls Shutdown.
func (m *Manager) Listen() {
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		m.Shutdown()
	}()
}

// Shutdown executes all registered handlers in reverse order.
// Safe to call multiple times; subsequent calls are no-ops.
func (m *Manager) Shutdown() {
	m.once.Do(func() {
		m.mu.Lock()
		handlers := make([]func() error, len(m.handlers))
		copy(handlers, m.handlers)
		m.mu.Unlock()

		for i := len(handlers) - 1; i >= 0; i-- {
			if err := handlers[i](); err != nil {
				log.Printf("[shutdown] handler error: %v", err)
			}
		}
	})
}

var (
	defaultManager *Manager
	defaultOnce    sync.Once
)

// DefaultManager returns the package-level shutdown manager,
// starting signal listening on first call.
func DefaultManager() *Manager {
	defaultOnce.Do(func() {
		defaultManager = NewManager()
		defaultManager.Listen()
	})
	return defaultManager
}

// OnShutdown registers fn with the default manager.
func OnShutdown(fn func() error) {
	DefaultManager().Register(fn)
}
