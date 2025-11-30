//go:build !use_quickjs

package jsruntime

import (
	"fmt"

	v8 "rogchap.com/v8go"
)

func init() {
	defaultRuntimeType = RuntimeV8
}

// newRuntime creates the default runtime for this build
func newRuntime() JSRuntime {
	return NewV8Runtime()
}

// V8Runtime wraps V8 for pooled usage
type V8Runtime struct {
	isolate *v8.Isolate
	context *v8.Context
}

// NewV8Runtime creates a new V8 runtime
func NewV8Runtime() *V8Runtime {
	isolate := v8.NewIsolate()
	context := v8.NewContext(isolate)
	return &V8Runtime{
		isolate: isolate,
		context: context,
	}
}

// Execute runs JavaScript code and returns the result
func (v *V8Runtime) Execute(code string) (string, error) {
	val, err := v.context.RunScript(code, "render.js")
	if err != nil {
		// Try to get more detailed error info
		if jsErr, ok := err.(*v8.JSError); ok {
			return "", fmt.Errorf("%s\n%s", jsErr.Message, jsErr.StackTrace)
		}
		return "", err
	}

	if val == nil {
		return "", nil
	}

	return val.String(), nil
}

// Reset prepares the runtime for reuse
// V8 contexts can accumulate state, so we recreate the context
func (v *V8Runtime) Reset() {
	if v.context != nil {
		v.context.Close()
	}
	v.context = v8.NewContext(v.isolate)
}

// Destroy permanently destroys the runtime
func (v *V8Runtime) Destroy() {
	if v.context != nil {
		v.context.Close()
		v.context = nil
	}
	if v.isolate != nil {
		v.isolate.Dispose()
		v.isolate = nil
	}
}
