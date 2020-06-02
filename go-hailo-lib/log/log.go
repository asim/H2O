package log

import (
	seelog "github.com/cihub/seelog"
)

var Source string

var defaultContext = NewContext()

// Tracef formats message according to format specifier
// and writes to default logger with log level = Trace.
func Tracef(format string, params ...interface{}) {
	defaultContext.Tracef(format, params...)
}

// Debugf formats message according to format specifier
// and writes to default logger with log level = Debug.
func Debugf(format string, params ...interface{}) {
	defaultContext.Debugf(format, params...)
}

// Infof formats message according to format specifier
// and writes to default logger with log level = Info.
func Infof(format string, params ...interface{}) {
	defaultContext.Infof(format, params...)
}

// Warnf formats message according to format specifier and writes to default logger with log level = Warn
func Warnf(format string, params ...interface{}) error {
	return defaultContext.Warnf(format, params...)
}

// Errorf formats message according to format specifier and writes to default logger with log level = Error
func Errorf(format string, params ...interface{}) error {
	return defaultContext.Errorf(format, params...)
}

// Criticalf formats message according to format specifier and writes to default logger with log level = Critical
func Criticalf(format string, params ...interface{}) error {
	return defaultContext.Criticalf(format, params...)
}

// Trace formats message using the default formats for its operands and writes to default logger with log level = Trace
func Trace(v ...interface{}) {
	defaultContext.Trace(v...)
}

// Debug formats message using the default formats for its operands and writes to default logger with log level = Debug
func Debug(v ...interface{}) {
	defaultContext.Debug(v...)
}

// Info formats message using the default formats for its operands and writes to default logger with log level = Info
func Info(v ...interface{}) {
	defaultContext.Info(v...)
}

// Warn formats message using the default formats for its operands and writes to default logger with log level = Warn
func Warn(v ...interface{}) error {
	return defaultContext.Warn(v...)
}

// Error formats message using the default formats for its operands and writes to default logger with log level = Error
func Error(v ...interface{}) error {
	return defaultContext.Error(v...)
}

// Critical formats message using the default formats for its operands and writes to default logger with log level = Critical
func Critical(v ...interface{}) error {
	return defaultContext.Critical(v...)
}

// Flush immediately processes all currently queued messages and all currently buffered messages.
// It is a blocking call which returns only after the queue is empty and all the buffers are empty.
//
// If Flush is called for a synchronous logger (type='sync'), it only flushes buffers (e.g. '<buffered>' receivers)
// , because there is no queue.
//
// Call this method when your app is going to shut down not to lose any log messages.
func Flush() {
	defaultContext.Flush()
}

func Disable() {
	seelog.ReplaceLogger(seelog.Disabled)
}
