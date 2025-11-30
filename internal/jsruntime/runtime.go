package jsruntime

import "sync"

// RuntimeType represents the type of JavaScript runtime
type RuntimeType string

const (
	RuntimeQuickJS RuntimeType = "quickjs"
	RuntimeV8      RuntimeType = "v8"
)

// defaultRuntimeType is set by init() in the build-specific files
var defaultRuntimeType RuntimeType

// JSRuntime is the interface for JavaScript execution
type JSRuntime interface {
	// Execute runs JavaScript code and returns the result as a string
	Execute(code string) (string, error)
	// Close releases resources (called when returning to pool)
	Reset()
	// Destroy permanently destroys the runtime
	Destroy()
}

// Pool manages a pool of JS runtimes for reuse
type Pool struct {
	runtimeType RuntimeType
	pool        chan JSRuntime
	maxSize     int
	created     int
	closed      bool
	mu          sync.Mutex

	// Track all created runtimes for proper cleanup
	allRuntimes []JSRuntime
	runtimesMu  sync.Mutex
}

// PoolConfig configures the runtime pool
type PoolConfig struct {
	RuntimeType RuntimeType
	PoolSize    int // Maximum number of runtimes to keep in pool
}

// DefaultRuntimeType returns the runtime type for this build
func DefaultRuntimeType() RuntimeType {
	return defaultRuntimeType
}

// NewPool creates a new runtime pool
func NewPool(config PoolConfig) *Pool {
	if config.PoolSize <= 0 {
		config.PoolSize = 10
	}
	// Use default runtime type if not specified
	if config.RuntimeType == "" {
		config.RuntimeType = defaultRuntimeType
	}

	p := &Pool{
		runtimeType: config.RuntimeType,
		maxSize:     config.PoolSize,
		pool:        make(chan JSRuntime, config.PoolSize),
		allRuntimes: make([]JSRuntime, 0, config.PoolSize),
	}

	// Pre-warm the pool
	for i := 0; i < config.PoolSize; i++ {
		rt := p.createRuntime()
		p.pool <- rt
	}

	return p
}

// createRuntime creates a new runtime and tracks it
func (p *Pool) createRuntime() JSRuntime {
	p.mu.Lock()
	p.created++
	p.mu.Unlock()

	rt := newRuntime()

	// Track for cleanup
	p.runtimesMu.Lock()
	p.allRuntimes = append(p.allRuntimes, rt)
	p.runtimesMu.Unlock()

	return rt
}

// Get retrieves a runtime from the pool
func (p *Pool) Get() JSRuntime {
	select {
	case rt := <-p.pool:
		return rt
	default:
		// Pool is empty, create a new one
		return p.createRuntime()
	}
}

// Put returns a runtime to the pool
func (p *Pool) Put(rt JSRuntime) {
	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		rt.Destroy()
		return
	}

	rt.Reset()
	select {
	case p.pool <- rt:
		// Successfully returned to pool
	default:
		// Pool is full, destroy the runtime
		rt.Destroy()
	}
}

// Execute is a convenience method that gets a runtime, executes code, and returns it
func (p *Pool) Execute(code string) (string, error) {
	rt := p.Get()
	defer p.Put(rt)
	return rt.Execute(code)
}

// Stats returns pool statistics
func (p *Pool) Stats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]interface{}{
		"runtime_type":  p.runtimeType,
		"total_created": p.created,
		"max_pool_size": p.maxSize,
		"pool_size":     len(p.pool),
		"closed":        p.closed,
	}
}

// Close marks the pool as closed and destroys all runtimes
func (p *Pool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	p.mu.Unlock()

	// Drain the pool
	close(p.pool)
	for rt := range p.pool {
		rt.Destroy()
	}

	// Destroy any remaining tracked runtimes
	p.runtimesMu.Lock()
	for _, rt := range p.allRuntimes {
		rt.Destroy()
	}
	p.allRuntimes = nil
	p.runtimesMu.Unlock()
}
