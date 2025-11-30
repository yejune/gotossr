//go:build use_quickjs

package jsruntime

import (
	"github.com/buke/quickjs-go"
)

func init() {
	defaultRuntimeType = RuntimeQuickJS
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewQuickJSRuntime()
}

// QuickJSRuntime wraps QuickJS for pooled usage
type QuickJSRuntime struct {
	runtime *quickjs.Runtime
	context *quickjs.Context
}

// NewQuickJSRuntime creates a new QuickJS runtime
func NewQuickJSRuntime() *QuickJSRuntime {
	rt := quickjs.NewRuntime()
	ctx := rt.NewContext()
	return &QuickJSRuntime{
		runtime: rt,
		context: ctx,
	}
}

// Execute runs JavaScript code and returns the result
func (q *QuickJSRuntime) Execute(code string) (string, error) {
	res := q.context.Eval(code)
	defer res.Free()

	if res.IsException() {
		return "", res.Error()
	}

	return res.String(), nil
}

// Reset prepares the runtime for reuse
// QuickJS contexts can accumulate state, so we recreate the context
func (q *QuickJSRuntime) Reset() {
	// Close old context and create a new one
	// This clears any global state from previous executions
	if q.context != nil {
		q.context.Close()
	}
	q.context = q.runtime.NewContext()
}

// Destroy permanently destroys the runtime
func (q *QuickJSRuntime) Destroy() {
	if q.context != nil {
		q.context.Close()
		q.context = nil
	}
	if q.runtime != nil {
		q.runtime.Close()
		q.runtime = nil
	}
}
