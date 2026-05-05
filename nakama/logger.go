package nakama

import (
	"fmt"
	"log"
)

// Logger is the interface used by the SDK to emit log messages.
// It mirrors Nakama/ILogger.cs from the .NET SDK.
type Logger interface {
	DebugFormat(format string, args ...any)
	Debug(args ...any)
	ErrorFormat(format string, args ...any)
	Error(args ...any)
	InfoFormat(format string, args ...any)
	Info(args ...any)
	WarnFormat(format string, args ...any)
	Warn(args ...any)
}

// NullLogger discards all log messages.
type NullLogger struct{}

func (NullLogger) DebugFormat(format string, args ...any) {}
func (NullLogger) Debug(args ...any)                      {}
func (NullLogger) ErrorFormat(format string, args ...any) {}
func (NullLogger) Error(args ...any)                      {}
func (NullLogger) InfoFormat(format string, args ...any)  {}
func (NullLogger) Info(args ...any)                       {}
func (NullLogger) WarnFormat(format string, args ...any)  {}
func (NullLogger) Warn(args ...any)                       {}

// StdLogger is a Logger backed by the standard library log package.
type StdLogger struct {
	Logger *log.Logger
}

// NewStdLogger creates a logger that writes through the supplied *log.Logger.
// If logger is nil, the default standard logger is used.
func NewStdLogger(logger *log.Logger) *StdLogger {
	return &StdLogger{Logger: logger}
}

func (s *StdLogger) write(level, msg string) {
	line := fmt.Sprintf("[%s] %s", level, msg)
	if s.Logger == nil {
		log.Println(line)
		return
	}
	s.Logger.Println(line)
}

func (s *StdLogger) DebugFormat(format string, args ...any) {
	s.write("DEBUG", fmt.Sprintf(format, args...))
}
func (s *StdLogger) Debug(args ...any) { s.write("DEBUG", fmt.Sprint(args...)) }
func (s *StdLogger) ErrorFormat(format string, args ...any) {
	s.write("ERROR", fmt.Sprintf(format, args...))
}
func (s *StdLogger) Error(args ...any) { s.write("ERROR", fmt.Sprint(args...)) }
func (s *StdLogger) InfoFormat(format string, args ...any) {
	s.write("INFO", fmt.Sprintf(format, args...))
}
func (s *StdLogger) Info(args ...any) { s.write("INFO", fmt.Sprint(args...)) }
func (s *StdLogger) WarnFormat(format string, args ...any) {
	s.write("WARN", fmt.Sprintf(format, args...))
}
func (s *StdLogger) Warn(args ...any) { s.write("WARN", fmt.Sprint(args...)) }
