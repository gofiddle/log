# Log

Log is an easy to use golang logging library. It supports level based  and asynchronized logging.


## Getting Started
### Install
~~~
go get github.com/gofiddle/log
~~~

### Log to File
Create a logger which writes to a log file.
~~~ go
package main

import "github.com/gofiddle/log"

func main() {
	logger := log.NewFileLogger("/var/log", "gofiddle", log.LOG_LEVEL_INFO)
	defer logger.Close()
	logger.Trace("This is a trace message.")
	logger.Debug("This is a debug message.")
	logger.Info("Hello World!")
	logger.Warn("This is a warnning message.")
	logger.Error("This is an error message.")
}
~~~

### Log to an HTTP Server
Create a logger which writes to a http log server.
~~~ go
package main

import "github.com/gofiddle/log"

func main() {
	logger := log.NewHTTPLogger("http://example.com:8080/log", log.LOG_LEVEL_INFO)
	logger.Trace("This is a trace message.")
	logger.Debug("This is a debug message.")
	logger.Info("Hello World!")
	logger.Warn("This is a warnning message.")
	logger.Error("This is an error message.")
}
~~~

### Provide your own LogWriter
~~~ go
package main

import "github.com/gofiddle/log"

func main() {
	logger := log.New(os.Stdout, log.LOG_LEVEL_INFO)
	logger.Trace("This is a trace message.")
	logger.Debug("This is a debug message.")
	logger.Info("Hello World!")
	logger.Warn("This is a warnning message.")
	logger.Error("This is an error message.")
}
~~~


### Provide your own LogWriter and make it logging asynchronizedly
``` go
package main

import "github.com/gofiddle/log"

func main() {
	logger := log.New(NewAsyncLogWriter(os.Stdout), log.LOG_LEVEL_INFO)
	logger.Trace("This is a trace message.")
	logger.Debug("This is a debug message.")
	logger.Info("Hello World!")
	logger.Warn("This is a warnning message.")
	logger.Error("This is an error message.")
}
```

### Customize log format
By default, the logger will format the log message to something like this: "INFO: 2006-01-02T15:04:05 (UTC): log message...", you can customize the format by providing your own formatter after created the logger.

~~~ go
package main

import "github.com/gofiddle/log"

type MyLogFormatter struct {}

func (f *MyLogFormatter) Format(t time.Time, level int, message string) string {
	... customize the message
	return msg
}

func main() {
	logger := log.New(NewAsyncLogWriter(os.Stdout), log.LOG_LEVEL_INFO)
	logger.Trace("This is a trace message.")
	logger.Debug("This is a debug message.")
	logger.Info("Hello World!")
	logger.Warn("This is a warnning message.")
	logger.Error("This is an error message.")
}
~~~

## Author and Maintainer
* Tom Li <nklizhe@gmail.com>

## License
MIT License