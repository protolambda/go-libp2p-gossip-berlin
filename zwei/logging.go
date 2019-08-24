package zwei

import "log"

type Logger interface {
	Printf(format string, values ...interface{})
	SubLogger(prefix string) Logger
}

type DebugLogger struct {
	logger *log.Logger
}

func NewDebugLogger(l *log.Logger) *DebugLogger {
	return &DebugLogger{logger: l}
}

func (dl *DebugLogger) Printf(format string, values ...interface{}) {
	if dl.logger != nil {
		dl.logger.Printf(format, values...)
	}
	// no-op if nil, for ignoring it during benchmarking.
}

func (dl *DebugLogger) SubLogger(prefix string) Logger {
	if dl.logger == nil {
		return new(DebugLogger)
	} else {
		return &DebugLogger{
			logger: log.New(dl.logger.Writer(), dl.logger.Prefix()+" > "+prefix, dl.logger.Flags()),
		}
	}
}
