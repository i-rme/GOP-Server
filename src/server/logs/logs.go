package logs

import (
	"fmt"
	"log"
	"os"
	"pfg/src/server/config"
	"time"
)

var (
	//Log is used to store in filesystem
	Log *log.Logger
)

func init() {

	time := time.Now()
	timeformat := time.Format("2006.01.02_15.04.05") // Defines time format for file name

	var logpath = config.LogRoot + "/" + timeformat + ".log"
	var logfile, err = os.Create(logpath)

	if err != nil {
		panic(err) //TO-DO Handle error
	}
	Log = log.New(logfile, "", 0) // Defines the log
}

// write proceeds to write to log
func write(line string) {
	fmt.Println(line)
	Log.Println(line)
}

// WriteDebug proceeds to write debug messages to log
func WriteDebug(line string) {
	if config.LoggingLevel == "debug" {
		write("DEBUG " + line)
	}
}

// WriteBasic proceeds to write debug messages to log
func WriteBasic(line string) {
	if config.LoggingLevel == "basic" || config.LoggingLevel == "debug" {
		write("INFO " + line)
	}
}

// WriteError proceeds to write debug messages to log
func WriteError(line string) {
	write("ERROR " + line)
}

// WriteRequest proceeds to write a request in the log
func WriteRequest(remoteAddr, method, host, requestURI, proto string) {
	WriteBasic(`<< ` + remoteAddr + ` - - [` + time.Now().Truncate(0).String() + `] "` + method + ` ` + host + requestURI + ` ` + proto + `"`)
	addRequest()
}

// WriteResponse proceeds to write a response in the log
func WriteResponse(remoteAddr, localAddr, executionTime string) {
	WriteDebug(`>> ` + remoteAddr + " was served by " + localAddr + " in " + executionTime)
}
