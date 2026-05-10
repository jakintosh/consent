package tokens

import "sync"

// DecodeLogFunc receives token validation failure context when decode logging is enabled.
type DecodeLogFunc func(context string)

var decodeLogger struct {
	mu sync.RWMutex
	fn DecodeLogFunc
}

// SetDecodeLogger configures optional logging for token decode validation failures.
// Passing nil disables logging. Decode logging is disabled by default.
func SetDecodeLogger(fn DecodeLogFunc) {
	decodeLogger.mu.Lock()
	defer decodeLogger.mu.Unlock()
	decodeLogger.fn = fn
}

func logDecodeError(err *validateError) {
	decodeLogger.mu.RLock()
	fn := decodeLogger.fn
	decodeLogger.mu.RUnlock()
	if fn != nil {
		fn(err.Context())
	}
}
