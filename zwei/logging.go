package zwei

import "log"

// Simple hackathon logger nesting
type Logger interface {
	Printf(format string, values ...interface{})
	SubLogger(prefix string) Logger
}

type DebugLogger struct {
	prefix string
	logger *log.Logger
}

func NewDebugLogger(l *log.Logger) *DebugLogger {
	return &DebugLogger{logger: l}
}

func (dl *DebugLogger) Printf(format string, values ...interface{}) {
	if dl.logger != nil {
		dl.logger.Printf(dl.prefix + ": " + format, values...)
	}
	// no-op if nil, for ignoring it during benchmarking.
}

func (dl *DebugLogger) SubLogger(prefix string) Logger {
	if dl.logger == nil {
		return new(DebugLogger)
	} else {
		return &DebugLogger{
			prefix: dl.prefix+prefix,
			logger: dl.logger,
		}
	}
}
