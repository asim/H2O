package log

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cihub/seelog"
)

type Context struct {
	fmt string
}

func NewContext(topics ...string) *Context {
	seelog.Current.SetAdditionalStackDepth(1)

	if len(topics) == 0 {
		return &Context{}
	}

	return &Context{
		fmt: "#" + strings.Join(topics, " #"),
	}

}

func (c *Context) NewContext(topics ...string) *Context {

	if len(topics) == 0 {
		return &Context{
			fmt: c.fmt,
		}
	}

	if c.fmt == "" {
		return &Context{
			fmt: "#" + strings.Join(topics, " #"),
		}
	}

	return &Context{
		fmt: c.fmt + " #" + strings.Join(topics, " #"),
	}
}

func (c *Context) AddContext(topics ...string) {
	*c = *(c.NewContext(topics...))
}

func (c *Context) prefixTopics(f string) string {
	switch {
	case Source == "" && c.fmt == "":
		return f
	case Source != "" && c.fmt == "":
		return fmt.Sprintf("@%s %s", Source, f)
	case Source == "" && c.fmt != "":
		return fmt.Sprintf("%s %s", c.fmt, f)
	default:
		return fmt.Sprintf("@%s %s %s", Source, c.fmt, f)
	}
}

// Tracef formats message according to format specifier
// and writes to default logger with log level = Trace.
func (c *Context) Tracef(format string, params ...interface{}) {
	seelog.Tracef(c.prefixTopics(format), params...)
}

// Debugf formats message according to format specifier
// and writes to default logger with log level = Debug.
func (c *Context) Debugf(format string, params ...interface{}) {
	seelog.Debugf(c.prefixTopics(format), params...)
}

// Infof formats message according to format specifier
// and writes to default logger with log level = Info.
func (c *Context) Infof(format string, params ...interface{}) {
	seelog.Infof(c.prefixTopics(format), params...)
}

// Warnf formats message according to format specifier and writes to default logger with log level = Warn
// Returns an error without the context decoration
func (c *Context) Warnf(format string, params ...interface{}) error {
	seelog.Warnf(c.prefixTopics(format), params...)
	return fmt.Errorf(format, params...)
}

// Errorf formats message according to format specifier and writes to default logger with log level = Error
// Returns an error without the context decoration
func (c *Context) Errorf(format string, params ...interface{}) error {
	seelog.Errorf(c.prefixTopics(format), params...)
	return fmt.Errorf(format, params...)
}

// Criticalf formats message according to format specifier and writes to default logger with log level = Critical
// Returns an error without the context decoration
func (c *Context) Criticalf(format string, params ...interface{}) error {
	seelog.Criticalf(c.prefixTopics(format), params...)
	return fmt.Errorf(format, params...)
}

// Trace formats message using the default formats for its operands and writes to default logger with log level = Trace
func (c *Context) Trace(v ...interface{}) {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Trace(v1...)
}

// Debug formats message using the default formats for its operands and writes to default logger with log level = Debug
func (c *Context) Debug(v ...interface{}) {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Debug(v1...)
}

// Info formats message using the default formats for its operands and writes to default logger with log level = Info
func (c *Context) Info(v ...interface{}) {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Info(v1...)
}

// Warn formats message using the default formats for its operands and writes to default logger with log level = Warn
// Returns an error without the context decoration
func (c *Context) Warn(v ...interface{}) error {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Warn(v1...)
	return errors.New(fmt.Sprint(v...))
}

// Error formats message using the default formats for its operands and writes to default logger with log level = Error
// Returns an error without the context decoration
func (c *Context) Error(v ...interface{}) error {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Error(v1...)
	return errors.New(fmt.Sprint(v...))
}

// Critical formats message using the default formats for its operands and writes to default logger with log level = Critical
// Returns an error without the context decoration
func (c *Context) Critical(v ...interface{}) error {
	v1 := []interface{}{c.prefixTopics("")}
	v1 = append(v1, v...)
	seelog.Critical(v1...)
	return errors.New(fmt.Sprint(v...))
}

// Call this method when your app is going to shut down not to lose any log messages.
func (c *Context) Flush() {
	seelog.Flush()
}
