package logger

import (
	"io"
	"os"
	"strings"
)

var (
	//DefaultOutput is used as the default log output.
	DefaultOutput io.Writer = os.Stderr

	// DefaultLevel is used as the default log level.
	DefaultLevel = Info
)

// Format is a simple convience type for when formatting is required. When
// processing a value of this type, the logger automatically treats the first
// argument as a Printf formatting string and passes the rest as the values
// to be formatted. For example: L.Info(Fmt{"%d beans/day", beans}).
type Format []interface{}

// Fmt returns a Format type. This is a convience function for creating a Format
// type.
func Fmt(str string, args ...interface{}) Format {
	return append(Format{str}, args...)
}

// A simple shortcut to format numbers in hex when displayed with the normal
// text output. For example: L.Info("header value", Hex(17))
type Hex int

// A simple shortcut to format numbers in octal when displayed with the normal
// text output. For example: L.Info("perms", Octal(17))
type Octal int

// A simple shortcut to format numbers in binary when displayed with the normal
// text output. For example: L.Info("bits", Binary(17))
type Binary int

// Level represents a log level.
type Level int32

const (
	// NoLevel is a special level used to indicate that no level has been
	// set and allow for a default to be used.
	NoLevel Level = 0

	// Trace is the most verbose level. Intended to be used for the tracing
	// of actions in code, such as function enters/exits, etc.
	Trace Level = 1

	// Debug information for programmer lowlevel analysis.
	Debug Level = 2

	// Info information about steady state operations.
	Info Level = 3

	// Warn information about rare but handled events.
	Warn Level = 4

	// Error information about unrecoverable events.
	Error Level = 5
)

// ColorOption defines color option
type ColorOption uint8

const (
	// ColorOff is the default coloration, and does not
	// inject color codes into the io.Writer.
	ColorOff ColorOption = iota
	// AutoColor checks if the io.Writer is a tty,
	// and if so enables coloring.
	AutoColor
	// ForceColor will enable coloring, regardless of whether
	// the io.Writer is a tty or not.
	ForceColor
)

// LevelFromString returns a Level type for the named log level, or "NoLevel" if
// the level string is invalid. This facilitates setting the log level via
// config or environment variable by name in a predictable way.
func LevelFromString(levelStr string) Level {
	// We don't care about case. Accept both "INFO" and "info".
	levelStr = strings.ToLower(strings.TrimSpace(levelStr))
	switch levelStr {
	case "trace":
		return Trace
	case "debug":
		return Debug
	case "info":
		return Info
	case "warn":
		return Warn
	case "error":
		return Error
	default:
		return NoLevel
	}
}

func (l Level) String() string {
	switch l {
	case Trace:
		return "trace"
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warn:
		return "warn"
	case Error:
		return "error"
	case NoLevel:
		return "none"
	default:
		return "unknown"
	}
}

// Logger describes the interface that must be implemeted by all loggers.
type Logger interface {
	// Args are alternating key, val pairs
	// keys must be strings
	// vals can be any type, but display is implementation specific
	// Emit a message and key/value pairs at a provided log level
	Log(level Level, msg string, args ...interface{})

	// Emit a message and key/value pairs at the TRACE level
	Trace(msg string, args ...interface{})

	// Emit a message and key/value pairs at the DEBUG level
	Debug(msg string, args ...interface{})

	// Emit a message and key/value pairs at the INFO level
	Info(msg string, args ...interface{})

	// Emit a message and key/value pairs at the WARN level
	Warn(msg string, args ...interface{})

	// Emit a message and key/value pairs at the ERROR level
	Error(msg string, args ...interface{})

	// Emit a message and key/value pairs at the ERROR level & panic
	Panic(msg string, args ...interface{})

	// If err not null Emit a message and panic
	ErrorPanic(err error, args ...interface{})

	// Indicate if TRACE logs would be emitted. This and the other Is* guards
	// are used to elide expensive logging code based on the current level.
	IsTrace() bool

	// Indicate if DEBUG logs would be emitted. This and the other Is* guards
	IsDebug() bool

	// Indicate if INFO logs would be emitted. This and the other Is* guards
	IsInfo() bool

	// Indicate if WARN logs would be emitted. This and the other Is* guards
	IsWarn() bool

	// Indicate if ERROR logs would be emitted. This and the other Is* guards
	IsError() bool

	// ImpliedArgs returns With key/value pairs
	ImpliedArgs() []interface{}

	// Creates a sublogger that will always have the given key/value pairs
	With(args ...interface{}) Logger

	// Returns the Name of the logger
	Name() string

	// Create a logger that will prepend the name string on the front of all messages.
	// If the logger already has a name, the new value will be appended to the current
	// name. That way, a major subsystem can use this to decorate all it's own logs
	// without losing context.
	Named(name string) Logger

	// Create a logger that will prepend the name string on the front of all messages.
	// This sets the name of the logger to the value directly, unlike Named which honor
	// the current name as well.
	ResetNamed(name string) Logger

	// Updates the level. This should affect all sub-loggers as well. If an
	// implementation cannot update the level on the fly, it should no-op.
	SetLevel(level Level)
}

// LoggerOptions can be used to configure a new logger.
type LoggerOptions struct {
	// Name of the subsystem to prefix logs with
	Name string

	// The threshold for the logger. Anything less severe is supressed
	Level Level

	// Where to write the logs to. Defaults to os.Stderr if nil
	Output []io.Writer

	// An optional Locker in case Output is shared. This can be a sync.Mutex or
	// a NoopLocker if the caller wants control over output, e.g. for batching
	// log lines.
	Mutex Locker

	// Control if the output should be in JSON.
	JSONFormat bool

	// Include file and line information in each log line
	IncludeLocation bool

	// The time format to use instead of the default
	TimeFormat string

	// Control whether or not to display the time at all. This is required
	// because setting TimeFormat to empty assumes the default format.
	DisableTime bool

	// Color the output. On Windows, colored logs are only avaiable for io.Writers that
	// are concretely instances of *os.File.
	Color []ColorOption

	// A function which is called with the log information and if it returns true the value
	// should not be logged.
	// This is useful when interacting with a system that you wish to suppress the log
	// message for (because it's too noisy, etc)
	Exclude func(level Level, msg string, args ...interface{}) bool
}

// Locker is used for locking output. If not set when creating a logger, a
// sync.Mutex will be used internally.
type Locker interface {
	// Lock is called when the output is going to be changed or written to
	Lock()

	// Unlock is called when the operation that called Lock() completes
	Unlock()
}

// Flushable represents a method for flushing an output buffer. It can be used
// if Resetting the log to use a new output, in order to flush the writes to
// the existing output beforehand.
type Flushable interface {
	Flush() error
}

// OutputResettable provides ways to swap the output in use at runtime
type OutputResettable interface {
	// ResetOutput swaps the current output writer with the one given in the
	// opts. Color options given in opts will be used for the new output.
	ResetOutput(opts *LoggerOptions) error

	// ResetOutputWithFlush swaps the current output writer with the one given
	// in the opts, first calling Flush on the given Flushable. Color options
	// given in opts will be used for the new output.
	ResetOutputWithFlush(opts *LoggerOptions, flushable Flushable) error
}

// NoopLocker implements locker but does nothing. This is useful if the client
// wants tight control over locking, in order to provide grouping of log
// entries or other functionality.
type NoopLocker struct{}

// Lock does nothing
func (n NoopLocker) Lock() {}

// Unlock does nothing
func (n NoopLocker) Unlock() {}

var _ Locker = (*NoopLocker)(nil)
