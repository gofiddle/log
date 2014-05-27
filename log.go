// Package log provide an easy to use logging package that supports level-based and asynchronized logging.
// It's designed to be used as a drop-in replacement of the standard log package
package log

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	_ = iota
	LOG_LEVEL_TRACE
	LOG_LEVEL_DEBUG
	LOG_LEVEL_INFO
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	LOG_LEVEL_FATAL
)

type HTTPLogWriter struct {
	url string
}

func (w *HTTPLogWriter) Write(data []byte) (n int, err error) {
	resp, err := http.Post(w.url, "html/text", bytes.NewReader(data))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// check response code
	if resp.StatusCode != http.StatusOK {
		err = errors.New(fmt.Sprintf("HTTPLogWriter: %s error!", resp.StatusCode))
		return 0, err
	}
	return len(data), err
}

type LogMessage struct {
	data []byte
}

const DEFAULT_QUEUE_SIZE = 100

type AsyncLogWriter struct {
	w      io.Writer
	queue  chan LogMessage
	closed chan int
}

func NewAsyncLogWriter(w io.Writer, n int) *AsyncLogWriter {
	if n <= 0 {
		n = DEFAULT_QUEUE_SIZE
	}
	queue := make(chan LogMessage, n)

	aw := &AsyncLogWriter{
		queue:  queue,
		w:      w,
		closed: make(chan int),
	}

	go func(w *AsyncLogWriter) {
		// process all queued messages until the queue is closed
		for msg := range w.queue {
			_, err := w.w.Write(msg.data)
			if err != nil {
				// the writer failed to write the message somehow,
				// we just discard the message here, but other implementations
				// might try to resend the message
			}
		}
		w.closed <- 1 // all messages are processed. ready to close
	}(aw)

	return aw
}

// Close closes the AsyncLogWriter. It will block here until the log message queue is drained.
func (w *AsyncLogWriter) Close() {
	close(w.queue)
	<-w.closed
}

func (w *AsyncLogWriter) Write(data []byte) (n int, err error) {
	w.queue <- LogMessage{data: data}
	return len(data), nil
}

type LogFormatter interface {
	Format(t time.Time, level int, message string) string
}

type Logger struct {
	mutex       *sync.Mutex
	level       int
	path        string
	fname       string
	writer      io.Writer
	writeCloser io.WriteCloser
	formatter   LogFormatter
}

// DefaultLogFormatter format log message in this format: "INFO: 2006-01-02T15:04:05 (UTC): log message..."
type DefaultLogFormatter struct {
}

func (f *DefaultLogFormatter) Format(t time.Time, level int, message string) string {
	timeStr := t.UTC().Format("2006-01-02T15:04:05 (MST)")
	return fmt.Sprintf("%s: %s: %s\n", LogLevel2String(level), timeStr, message)
}

// New creates a new logger with the given writer
func New(w io.Writer, loglevel int) *Logger {
	logger := Logger{
		level:     loglevel,
		writer:    w,
		formatter: &DefaultLogFormatter{},
		mutex:     &sync.Mutex{},
	}
	if wc, ok := w.(io.WriteCloser); ok {
		logger.writeCloser = wc
	}
	return &logger
}

// NewHTTPLogger creates a logger that sends log to a http server
func NewHTTPLogger(url string, loglevel int) *Logger {
	return &Logger{
		level:     loglevel,
		writer:    NewAsyncLogWriter(&HTTPLogWriter{url: url}, DEFAULT_QUEUE_SIZE),
		formatter: &DefaultLogFormatter{},
		mutex:     &sync.Mutex{},
	}
}

// NewFileLogger creates a new logger which writes logs to the specified logpath and filename
func NewFileLogger(logpath string, fname string, loglevel int) (logger *Logger, err error) {

	// create the log directory if not exists
	err = os.MkdirAll(logpath, 0750)
	if err != nil {
		return nil, err
	}

	// use program name as log filename
	if fname == "" {
		fname = path.Base(os.Args[0])
	}
	filepath := fmt.Sprintf("%s/%s.log", logpath, fname)

	// open the log file
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		return nil, err
	}

	return &Logger{
		level:       loglevel,
		path:        logpath,
		fname:       fname,
		writeCloser: file,
		writer:      file,
		formatter:   &DefaultLogFormatter{},
		mutex:       &sync.Mutex{},
	}, nil
}

// SetLogLevel sets the current log level of the logger
func (logger *Logger) SetLogLevel(level int) {
	logger.level = level
}

// SetFormater sets the current formater to the new one
func (logger *Logger) SetFormatter(formatter LogFormatter) {
	logger.mutex.Lock()
	logger.formatter = formatter
	logger.mutex.Unlock()
}

// Close closes logger. If the log writer implements the io.WriteCloser interface, the logger will close the writer too.
func (logger *Logger) Close() {
	logger.mutex.Lock()
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	logger.mutex.Unlock()
}

// Writer returns current writer of the logger.
func (logger *Logger) Writer() io.Writer {
	return logger.writer
}

func (logger *Logger) Format(t time.Time, level int, message string) string {
	var msg string
	logger.mutex.Lock()
	if logger.formatter != nil {
		msg = logger.formatter.Format(t, level, message)
	}
	logger.mutex.Unlock()
	return msg
}

// Print logs a formatted message at LOG_LEVEL_INFO level
func (logger *Logger) Print(v ...interface{}) {
	s := fmt.Sprint(v...)
	msg := logger.Format(time.Now(), logger.level, s)
	if logger.Writer() != nil {
		logger.Writer().Write([]byte(msg))
	}
}

// Println logs a formatted message at LOG_LEVEL_INFO level
func (logger *Logger) Println(v ...interface{}) {
	s := fmt.Sprintln(v...)
	msg := logger.Format(time.Now(), logger.level, s)
	if logger.Writer() != nil {
		logger.Writer().Write([]byte(msg))
	}
}

// Println logs a formatted message at LOG_LEVEL_INFO level
func (logger *Logger) Printf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	msg := logger.Format(time.Now(), logger.level, s)
	if logger.Writer() != nil {
		logger.Writer().Write([]byte(msg))
	}
}

// Log logs a formatted message at the given log level
func (logger *Logger) Log(loglevel int, v ...interface{}) {
	if loglevel >= logger.level {
		s := fmt.Sprint(v...)
		msg := logger.Format(time.Now(), loglevel, s)
		if logger.Writer() != nil {
			logger.Writer().Write([]byte(msg))
		}
	}
}

// Logf logs a formatted message at the given log level
func (logger *Logger) Logf(loglevel int, format string, v ...interface{}) {
	if loglevel >= logger.level {
		s := fmt.Sprintf(format, v...)
		msg := logger.Format(time.Now(), loglevel, s)
		if logger.Writer() != nil {
			logger.Writer().Write([]byte(msg))
		}
	}
}

// Logln logs a formatted message at the given log level
func (logger *Logger) Logln(loglevel int, v ...interface{}) {
	if loglevel >= logger.level {
		s := fmt.Sprintln(v...)
		msg := logger.Format(time.Now(), loglevel, s)
		if logger.Writer() != nil {
			logger.Writer().Write([]byte(msg))
		}
	}
}

// Trace logs a formatted message at log level: LOG_LEVEL_TRACE
func (logger *Logger) Trace(v ...interface{}) {
	logger.Log(LOG_LEVEL_TRACE, v...)
}

// Tracef logs a formatted message at log level: LOG_LEVEL_TRACE
func (logger *Logger) Tracef(fmt string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_TRACE, fmt, v...)
}

// Tracef logs a formatted message at log level: LOG_LEVEL_TRACE
func (logger *Logger) Traceln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_TRACE, v...)
}

// Debug logs a formatted message at log level: LOG_LEVEL_DEBUG
func (logger *Logger) Debug(v ...interface{}) {
	logger.Log(LOG_LEVEL_DEBUG, v...)
}

// Debugf logs a formatted message at log level: LOG_LEVEL_DEBUG
func (logger *Logger) Debugf(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_DEBUG, format, v...)
}

// Debugln logs a formatted message at log level: LOG_LEVEL_DEBUG
func (logger *Logger) Debugln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_DEBUG, v...)
}

// Info logs a formatted message at log level: LOG_LEVEL_INFO
func (logger *Logger) Info(v ...interface{}) {
	logger.Log(LOG_LEVEL_INFO, v...)
}

// Infof logs a formatted message at log level: LOG_LEVEL_INFO
func (logger *Logger) Infof(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_INFO, format, v...)
}

// Infoln logs a formatted message at log level: LOG_LEVEL_INFO
func (logger *Logger) Infoln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_INFO, v...)
}

// Warn logs a formatted message at log level: LOG_LEVEL_WARN
func (logger *Logger) Warn(v ...interface{}) {
	logger.Log(LOG_LEVEL_WARN, v...)
}

// Warnf logs a formatted message at log level: LOG_LEVEL_WARN
func (logger *Logger) Warnf(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_WARN, format, v...)
}

// Warnln logs a formatted message at log level: LOG_LEVEL_WARN
func (logger *Logger) Warnln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_WARN, v...)
}

// Error logs a formatted message at log level: LOG_LEVEL_ERROR
func (logger *Logger) Error(v ...interface{}) {
	logger.Log(LOG_LEVEL_ERROR, v...)
}

// Errorf logs a formatted message at log level: LOG_LEVEL_ERROR
func (logger *Logger) Errorf(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_ERROR, format, v...)
}

// Errorln logs a formatted message at log level: LOG_LEVEL_ERROR
func (logger *Logger) Errorln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_ERROR, v...)
}

// Fatal logs a formatted message at log level: LOG_LEVEL_FATAL then calls os.Exit(1)
func (logger *Logger) Fatal(v ...interface{}) {
	logger.Log(LOG_LEVEL_FATAL, v...)
	os.Exit(1)
}

// Fatalf logs a formatted message at log level: LOG_LEVEL_FATAL then calls os.Exit(1)
func (logger *Logger) Fatalf(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_FATAL, format, v...)
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	os.Exit(1)
}

// Panic logs a formatted message at log level: LOG_LEVEL_FATAL then calls os.Exit(1)
func (logger *Logger) Fatalln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_FATAL, v...)
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	os.Exit(1)
}

// Panic logs a message at log level: LOG_LEVEL_FATAL then calls panic()
func (logger *Logger) Panic(v ...interface{}) {
	logger.Log(LOG_LEVEL_FATAL, v...)
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	panic(nil)
}

// Panicf logs a formatted message at log level: LOG_LEVEL_FATAL then calls panic()
func (logger *Logger) Panicf(format string, v ...interface{}) {
	logger.Logf(LOG_LEVEL_FATAL, format, v...)
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	panic(nil)
}

// Panicln logs a formatted message at log level: LOG_LEVEL_FATAL then calls panic()
func (logger *Logger) Panicln(v ...interface{}) {
	logger.Logln(LOG_LEVEL_FATAL, v...)
	if logger.writeCloser != nil {
		logger.writeCloser.Close()
	}
	panic(nil)
}

// LogLevel2String returns the string format of the given loglevel enum
func LogLevel2String(level int) string {
	switch level {
	case LOG_LEVEL_TRACE:
		return "TRACE"
	case LOG_LEVEL_DEBUG:
		return "DEBUG"
	case LOG_LEVEL_INFO:
		return "INFO"
	case LOG_LEVEL_WARN:
		return "WARN"
	case LOG_LEVEL_ERROR:
		return "ERROR"
	case LOG_LEVEL_FATAL:
		return "FATAL"
	default:
		return "Unknown"
	}
}

func String2LogLevel(str string) int {
	str = strings.ToUpper(str)
	switch str {
	case "TRACE":
		return LOG_LEVEL_TRACE
	case "DEBUG":
		return LOG_LEVEL_DEBUG
	case "INFO":
		return LOG_LEVEL_INFO
	case "WARN":
		return LOG_LEVEL_WARN
	case "ERROR":
		return LOG_LEVEL_WARN
	case "FATAL":
		return LOG_LEVEL_FATAL
	default:
		return -1
	}
}
