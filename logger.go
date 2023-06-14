package swe

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type LogLevel int

const (
	LOG_DEBUG LogLevel = iota
	LOG_INFO
	LOG_WARN
	LOG_ERROR
)

func (l LogLevel) String() string {
	switch l {
	case LOG_DEBUG:
		return "DEBUG"
	case LOG_INFO:
		return "INFO"
	case LOG_WARN:
		return "WARN"
	case LOG_ERROR:
		return "ERROR"
	}
	return "UNKNOWN_" + fmt.Sprint(int(l))
}

var defaultLogLevel LogLevel = LOG_DEBUG

func SetDefaultLogLevel(level LogLevel) {
	defaultLogLevel = level
}

// -----------------------------------------------------------------------------

func RenderTime(ts time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d.%03d",
		ts.Year(), ts.Month(), ts.Day(),
		ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond()/1000000)
}

// -----------------------------------------------------------------------------

type LogRenderer interface {
	RenderLog(ctx *Context, level LogLevel, ts time.Time, file string, line int, content string) string
}

type plainLogRenderer struct{}

func (r plainLogRenderer) RenderLog(ctx *Context, level LogLevel, ts time.Time, file string, line int, content string) string {
	return fmt.Sprintf("[%s][%s][%s:%d] %s\n", level.String(), RenderTime(ts), filepath.Base(file), line, content)
}

var defaultLogRenderer LogRenderer = plainLogRenderer{}

func SetDefaultLogRenderer(r LogRenderer) {
	if r != nil {
		defaultLogRenderer = r
	}
}

// -----------------------------------------------------------------------------

var defaultLogWriter io.Writer = os.Stdout

func SetDefaultLogWriter(out io.Writer) {
	defaultLogWriter = out
}

// -----------------------------------------------------------------------------

type Logger struct {
	ctx   *Context
	r     LogRenderer
	level LogLevel
	out   io.Writer
}

func CtxLogger(ctx *Context) *Logger {
	if ctx == nil {
		return &Logger{
			ctx:   ctx,
			r:     defaultLogRenderer,
			level: defaultLogLevel,
			out:   defaultLogWriter,
		}
	}

	key := "__engine_logger"
	ret, ok := CtxValue[*Logger](ctx, key)
	if ok {
		return ret
	}
	ret = &Logger{
		ctx:   ctx,
		r:     defaultLogRenderer,
		level: defaultLogLevel,
		out:   defaultLogWriter,
	}
	ctx.Put(key, ret)
	return ret
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) SetRenderer(r LogRenderer) {
	l.r = r
}

func (l *Logger) SetWriter(out io.Writer) {
	l.out = out
}

func (l *Logger) doLog(level LogLevel, format string, params ...any) {
	_, file, line, _ := runtime.Caller(3)
	l.out.Write([]byte(l.r.RenderLog(l.ctx, level, time.Now(), file, line, fmt.Sprintf(format, params...))))
}

func (l *Logger) tryLog(level LogLevel, format string, params ...any) {
	if l.level <= level {
		l.doLog(level, format, params...)
	}
}

func (l *Logger) Debug(format string, params ...any) {
	l.tryLog(LOG_DEBUG, format, params...)
}

func (l *Logger) Info(format string, params ...any) {
	l.tryLog(LOG_INFO, format, params...)
}

func (l *Logger) Warn(format string, params ...any) {
	l.tryLog(LOG_WARN, format, params...)
}

func (l *Logger) Error(format string, params ...any) {
	l.tryLog(LOG_ERROR, format, params...)
}
