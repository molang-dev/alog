// Package alog provides a small leveled logger with console output, daily file
// output, package-level helper functions, and independent logger instances.
package alog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Level controls log filtering.
//
// Levels are ordered from low to high:
//
//	LevelVerbose < LevelDebug < LevelInfo < LevelWarning < LevelError < LevelFatal
//
// SetLevel filters out logs below the configured level. Error and fatal logs are
// always printed, even when the configured level is higher than them.
type Level int

const (
	// LevelVerbose is the most detailed level and is printed in white on screen.
	LevelVerbose Level = iota

	// LevelDebug is intended for development diagnostics and is printed in blue.
	LevelDebug

	// LevelInfo is intended for normal runtime information and is printed in green.
	LevelInfo

	// LevelWarning is intended for recoverable problems and is printed in yellow.
	LevelWarning

	// LevelError is intended for errors and is printed in red.
	LevelError

	// LevelFatal is intended for fatal errors and is printed in red.
	LevelFatal
)

// Output flags control where logs are written and whether screen output is
// colorized. Flags can be combined with bitwise OR.
const (
	// FlagScreen writes logs to the logger output writer.
	FlagScreen = 1 << iota

	// FlagFile writes logs to a daily file named with the current local date,
	// for example 2026-06-27.log. Existing files are appended to.
	FlagFile

	// FlagColor wraps screen output in ANSI color escape sequences.
	FlagColor
)

// CallerFlag controls which caller fields can be added to a log line.
//
// Caller fields are disabled by default because they require runtime.Caller.
// Use SetCallerFlags to enable them only at or above a chosen level.
type CallerFlag int

const (
	// FlagShortFile adds the caller file base name and line, for example
	// main.go:23.
	FlagShortFile CallerFlag = 1 << iota

	// FlagLongFile adds the caller full file path and line, for example
	// /app/main.go:23.
	FlagLongFile

	// FlagFunc adds the caller function name, for example main.main.
	FlagFunc
)

const (
	defaultFlags = FlagScreen | FlagColor
	timeLayout   = "2006-01-02 15:04:05.000"
	dateLayout   = "2006-01-02"
	callerSkip   = 4
)

// Logger is the public logging interface implemented by loggers created with
// New.
type Logger interface {
	// V writes a verbose log.
	V(tag string, format string, msg ...any)

	// D writes a debug log.
	D(tag string, format string, msg ...any)

	// I writes an info log.
	I(tag string, format string, msg ...any)

	// W writes a warning log.
	W(tag string, format string, msg ...any)

	// E writes an error log. Error logs cannot be filtered by SetLevel.
	E(tag string, format string, msg ...any)

	// Fatal writes a fatal log, writes the current stack, and exits the process
	// with status code 1.
	Fatal(tag string, format string, msg ...any)

	// Time records a start time and returns its id.
	Time() (id uint32)

	// TimeEnd writes a debug log with the elapsed time since Time returned id.
	TimeEnd(id uint32, tag string, format string, msg ...any)

	// SetOutput changes the writer used by FlagScreen. Passing nil discards
	// screen output.
	SetOutput(w io.Writer)

	// Flags returns the current output flags.
	Flags() int

	// SetFlags replaces the current output flags.
	SetFlags(flag int)

	// Prefix returns the current optional prefix.
	Prefix() string

	// SetPrefix changes the optional prefix. Empty prefixes are omitted from the
	// rendered log line and do not occupy an empty "|" field.
	SetPrefix(prefix string)

	// Level returns the current filtering level.
	Level() Level

	// SetLevel changes the filtering level.
	SetLevel(level Level)

	// CallerFlags returns the minimum level and caller flags configured by
	// SetCallerFlags.
	CallerFlags() (Level, CallerFlag)

	// SetCallerFlags enables caller fields for logs at level or above. For
	// example, SetCallerFlags(LevelWarning, FlagShortFile) adds caller fields to
	// warning, error, and fatal logs only.
	SetCallerFlags(level Level, flags CallerFlag)
}

type logger struct {
	mu          sync.Mutex
	output      io.Writer
	flags       int
	prefix      string
	level       Level
	callerLevel Level
	callerFlags CallerFlag
	fileDate    string
	file        *os.File
	times       map[uint32]time.Time
	nextTimeID  uint32
}

var std = newLogger()

// New creates an independent logger instance.
func New() Logger {
	return newLogger()
}

// V writes a verbose log with the package-level default logger.
func V(tag string, format string, msg ...any) {
	std.log(LevelVerbose, tag, format, msg...)
}

// D writes a debug log with the package-level default logger.
func D(tag string, format string, msg ...any) {
	std.log(LevelDebug, tag, format, msg...)
}

// I writes an info log with the package-level default logger.
func I(tag string, format string, msg ...any) {
	std.log(LevelInfo, tag, format, msg...)
}

// W writes a warning log with the package-level default logger.
func W(tag string, format string, msg ...any) {
	std.log(LevelWarning, tag, format, msg...)
}

// E writes an error log with the package-level default logger.
func E(tag string, format string, msg ...any) {
	std.log(LevelError, tag, format, msg...)
}

// Fatal writes a fatal log with the package-level default logger, writes the
// current stack, and exits the process with status code 1.
func Fatal(tag string, format string, msg ...any) {
	std.log(LevelFatal, tag, format, msg...)
	std.writeStack()
	os.Exit(1)
}

// Time records a start time with the package-level default logger.
func Time() (id uint32) {
	return std.Time()
}

// TimeEnd writes a debug elapsed-time log with the package-level default logger.
func TimeEnd(id uint32, tag string, format string, msg ...any) {
	std.TimeEnd(id, tag, format, msg...)
}

// SetOutput changes the screen writer of the package-level default logger.
func SetOutput(w io.Writer) {
	std.SetOutput(w)
}

// Flags returns the output flags of the package-level default logger.
func Flags() int {
	return std.Flags()
}

// SetFlags replaces the output flags of the package-level default logger.
func SetFlags(flag int) {
	std.SetFlags(flag)
}

// Prefix returns the optional prefix of the package-level default logger.
func Prefix() string {
	return std.Prefix()
}

// SetPrefix changes the optional prefix of the package-level default logger.
func SetPrefix(prefix string) {
	std.SetPrefix(prefix)
}

// GetLevel returns the filtering level of the package-level default logger.
//
// The name is GetLevel instead of Level because Go does not allow a package to
// declare both type Level and function Level.
func GetLevel() Level {
	return std.Level()
}

// SetLevel changes the filtering level of the package-level default logger.
func SetLevel(level Level) {
	std.SetLevel(level)
}

// CallerFlags returns the caller configuration of the package-level default
// logger.
func CallerFlags() (Level, CallerFlag) {
	return std.CallerFlags()
}

// SetCallerFlags changes the caller configuration of the package-level default
// logger.
func SetCallerFlags(level Level, flags CallerFlag) {
	std.SetCallerFlags(level, flags)
}

func newLogger() *logger {
	return &logger{
		output:      os.Stdout,
		flags:       defaultFlags,
		level:       LevelVerbose,
		callerLevel: LevelFatal,
		times:       make(map[uint32]time.Time),
	}
}

func (l *logger) V(tag string, format string, msg ...any) {
	l.log(LevelVerbose, tag, format, msg...)
}

func (l *logger) D(tag string, format string, msg ...any) {
	l.log(LevelDebug, tag, format, msg...)
}

func (l *logger) I(tag string, format string, msg ...any) {
	l.log(LevelInfo, tag, format, msg...)
}

func (l *logger) W(tag string, format string, msg ...any) {
	l.log(LevelWarning, tag, format, msg...)
}

func (l *logger) E(tag string, format string, msg ...any) {
	l.log(LevelError, tag, format, msg...)
}

func (l *logger) Fatal(tag string, format string, msg ...any) {
	l.log(LevelFatal, tag, format, msg...)
	l.writeStack()
	os.Exit(1)
}

func (l *logger) Time() (id uint32) {
	id = atomic.AddUint32(&l.nextTimeID, 1)

	l.mu.Lock()
	l.times[id] = time.Now()
	l.mu.Unlock()

	return id
}

func (l *logger) TimeEnd(id uint32, tag string, format string, msg ...any) {
	l.mu.Lock()
	start, ok := l.times[id]
	if ok {
		delete(l.times, id)
	}
	l.mu.Unlock()

	if !ok {
		l.D(tag, format, msg...)
		return
	}

	message := fmt.Sprintf(format, msg...)
	if message != "" {
		message += " "
	}
	message += "elapsed=" + time.Since(start).String()
	l.D(tag, "%s", message)
}

func (l *logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	if w == nil {
		l.output = io.Discard
	} else {
		l.output = w
	}
	l.mu.Unlock()
}

func (l *logger) Flags() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.flags
}

func (l *logger) SetFlags(flag int) {
	l.mu.Lock()
	l.flags = flag
	l.mu.Unlock()
}

func (l *logger) Prefix() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.prefix
}

func (l *logger) SetPrefix(prefix string) {
	l.mu.Lock()
	l.prefix = prefix
	l.mu.Unlock()
}

func (l *logger) Level() Level {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.level
}

func (l *logger) SetLevel(level Level) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

func (l *logger) CallerFlags() (Level, CallerFlag) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.callerLevel, l.callerFlags
}

func (l *logger) SetCallerFlags(level Level, flags CallerFlag) {
	l.mu.Lock()
	l.callerLevel = level
	l.callerFlags = flags
	l.mu.Unlock()
}

func (l *logger) log(level Level, tag string, format string, msg ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level && level < LevelError {
		return
	}

	line := l.formatLine(level, tag, format, msg...)
	if l.flags&FlagScreen != 0 {
		if l.flags&FlagColor != 0 {
			_, _ = fmt.Fprintln(l.output, level.color()+line+"\033[0m")
		} else {
			_, _ = fmt.Fprintln(l.output, line)
		}
	}
	if l.flags&FlagFile != 0 {
		if file := l.openFileLocked(time.Now()); file != nil {
			_, _ = fmt.Fprintln(file, line)
		}
	}
}

func (l *logger) formatLine(level Level, tag string, format string, msg ...any) string {
	now := time.Now().Format(timeLayout)
	message := fmt.Sprintf(format, msg...)
	pid := fmt.Sprintf("%d", os.Getpid())
	callerFields := l.callerFields(level)

	parts := []string{now, level.letter(), pid}
	if l.prefix != "" {
		parts = append(parts, l.prefix)
	}
	if tag != "" {
		parts = append(parts, tag)
	}
	parts = append(parts, callerFields...)
	if message != "" {
		parts = append(parts, message)
	}

	return strings.Join(parts, "|")
}

func (l *logger) callerFields(level Level) []string {
	if l.callerFlags == 0 || level < l.callerLevel {
		return nil
	}

	pc, file, line, ok := runtime.Caller(callerSkip)
	if !ok {
		return nil
	}

	fields := make([]string, 0, 2)
	if l.callerFlags&FlagLongFile != 0 {
		fields = append(fields, fmt.Sprintf("%s:%d", file, line))
	} else if l.callerFlags&FlagShortFile != 0 {
		fields = append(fields, fmt.Sprintf("%s:%d", filepath.Base(file), line))
	}
	if l.callerFlags&FlagFunc != 0 {
		if fn := runtime.FuncForPC(pc); fn != nil {
			fields = append(fields, fn.Name())
		}
	}

	return fields
}

func (l *logger) openFileLocked(now time.Time) *os.File {
	date := now.Format(dateLayout)
	if l.file != nil && l.fileDate == date {
		return l.file
	}

	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}

	file, err := os.OpenFile(date+".log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil
	}

	l.file = file
	l.fileDate = date
	return file
}

func (l *logger) writeStack() {
	l.mu.Lock()
	defer l.mu.Unlock()

	stack := strings.TrimRight(string(debug.Stack()), "\n")
	if l.flags&FlagScreen != 0 {
		if l.flags&FlagColor != 0 {
			_, _ = fmt.Fprintln(l.output, LevelFatal.color()+stack+"\033[0m")
		} else {
			_, _ = fmt.Fprintln(l.output, stack)
		}
	}
	if l.flags&FlagFile != 0 {
		if file := l.openFileLocked(time.Now()); file != nil {
			_, _ = fmt.Fprintln(file, stack)
		}
	}
}

func (l Level) letter() string {
	switch l {
	case LevelVerbose:
		return "V"
	case LevelDebug:
		return "D"
	case LevelInfo:
		return "I"
	case LevelWarning:
		return "W"
	case LevelError:
		return "E"
	case LevelFatal:
		return "F"
	default:
		return "I"
	}
}

func (l Level) color() string {
	switch l {
	case LevelVerbose:
		return "\033[37m"
	case LevelDebug:
		return "\033[34m"
	case LevelInfo:
		return "\033[32m"
	case LevelWarning:
		return "\033[33m"
	case LevelError, LevelFatal:
		return "\033[31m"
	default:
		return ""
	}
}
