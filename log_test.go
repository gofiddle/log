package log

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	fmt.Println("Running TestLogger...")

	// Create a logger that logs to Stdout
	logger := New(os.Stdout, LOG_LEVEL_DEBUG)

	// Print some log messages and they should appear in the file /tmp/log.test.log
	logger.Trace("This is a trace message") // This message shouldn't be logged
	logger.Debug("This is a debug message")
	logger.Info("This is a info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")
}

func TestFileLogger(t *testing.T) {
	fmt.Println("Running TestFileLogger...")

	// Create a logger that logs to /tmp
	// We don't specify the log filename, so it will automatically use the program name saved in os.Args[0]
	logger := NewFileLogger("/tmp", "", LOG_LEVEL_DEBUG)

	// We need to make sure the logger will be closed
	defer logger.Close()

	// Print some log messages and they should appear in the file /tmp/log.test.log
	logger.Trace("This is a trace message") // This message shouldn't be logged
	logger.Debug("This is a debug message")
	logger.Info("This is a info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")
}

func startLogServer() {
	server := http.Server{Addr: "127.0.0.1:8080", Handler: func() http.Handler {
		mux := http.NewServeMux()
		mux.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			data, err := ioutil.ReadAll(r.Body)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error handling message!", err.Error())
				http.Error(w, err.Error(), 400)
				return
			}
			fmt.Fprintf(os.Stdout, "%s: %s", r.RemoteAddr, string(data))
			w.Write([]byte("OK"))
		})
		return mux
	}()}
	go server.ListenAndServe()
}

func stopLogServerAfter(seconds time.Duration) {
	stop := make(chan int)
	go func() {
		<-time.After(time.Second * seconds)
		stop <- 1
	}()

	<-stop
}

func TestHTTPLogger(t *testing.T) {
	fmt.Println("Running TestHTTPLogger...")

	// Start HTTP Log Server
	startLogServer()

	// Create a logger that logs to the http log server
	logger := NewHTTPLogger("http://127.0.0.1:8080/log", LOG_LEVEL_DEBUG)

	// Print some log messages and they should appear in the file /tmp/log.test.log
	logger.Trace("This is a trace message") // This message shouldn't be logged
	logger.Debug("This is a debug message")
	logger.Info("This is a testing message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	// Wait for 5 seconds to make sure the messages have reached the server
	stopLogServerAfter(5)
}

func BenchmarkHTTPLogger(b *testing.B) {

	// Start HTTP Log Server
	startLogServer()

	// Create a logger that logs to the http log server
	logger := NewHTTPLogger("http://127.0.0.1:8080/log", LOG_LEVEL_DEBUG)

	// Sending 100 log messages to the http server should take no time
	for i := 0; i < 100; i++ {
		logger.Info("This is a testing message.")
	}

	// Wait for 5 seconds to make sure the messages have reached the server
	stopLogServerAfter(5)
}
