package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

var logger = log.New()

func init() {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		log.Info("Failed get current working directory")
		log.Fatal(err)
	}
	layout := "2006-01-02"
	env := os.Getenv("ENV")
	fmt.Println("ENV", env)
	formatTime := time.Now().Format(layout)
	// file, err := os.OpenFile(filepath.Join(cwd, fmt.Sprintf("%s%s-%s%s", "logs/", formatTime, env, ".log")), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if env == "stage" {
		logger.Out = os.Stdout
	}
	if env == "prod" || env == "" {
		file, err := os.OpenFile(
			filepath.Join(cwd, fmt.Sprintf("%s%s%s%s", "logs/", formatTime, env, ".log")),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0o666,
		)
		if err != nil {
			log.Info("Failed to log to file, using default stderr")
			// log.Fatal(err)
		}
		logger.Out = file
	}

	logger.Formatter = &log.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	logger.SetLevel(log.DebugLevel)
}

func GetLogger() *log.Entry {
	// The API for setting attributes is a little different than the package level
	// exported logger. See Godoc.
	// log.Out = os.Stdout

	// You could set this to any `io.Writer` such as a file
	// file, err := os.OpenFile("log.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	// if err == nil {
	//  log.Out = file
	// } else {
	//  log.Info("Failed to log to file, using default stderr")
	// }
	function, file, line, _ := runtime.Caller(1)

	functionObject := runtime.FuncForPC(function)
	entry := logger.WithFields(log.Fields{
		"requestId": time.Now().UnixNano() / int64(time.Millisecond),
		"size":      10,
		"function":  functionObject.Name(),
		"file":      file,
		"line":      line,
	})

	return entry
}
