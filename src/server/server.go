package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"pfg/src/handler"
	"pfg/src/repository/filesystem"
	"pfg/src/server/config"
	"pfg/src/server/logs"
	"time"
)

var lastTimestamp time.Time

func init() {
	filesystem.RemoveCache(config.CacheRoot)
	lastTimestamp = time.Now()
}

// stateCallback is called when the state of a request changes
func stateCallback(conn net.Conn, ConnState http.ConnState) {

	currentTime := time.Now()
	elapsedTime := time.Since(lastTimestamp)
	lastTimestamp = currentTime //Updates timestamp for subsequent requests
	timeSpent := fmt.Sprint(elapsedTime.Milliseconds()) + "ms"

	switch ConnState {
	case http.StateNew:
		//fmt.Println("StateNew")
	case http.StateActive:
		//fmt.Println("StateActive")
	case http.StateIdle:
		logs.WriteResponse(conn.RemoteAddr().String(), conn.LocalAddr().String(), timeSpent)
	case http.StateHijacked:
		//fmt.Println("StateHijacked")
	case http.StateClosed:
		//fmt.Println("StateClosed")
	}

}

// Start innitiates the server
func Start() {
	connectionString := fmt.Sprintf("%s%s%d", config.Address, ":", config.Port) //Formatting the connection, adding colon (:)

	server := &http.Server{
		Addr:           connectionString,
		Handler:        nil,
		ReadTimeout:    config.ReadTimeout,
		WriteTimeout:   config.WriteTimeout,
		IdleTimeout:    config.IdleTimeout,
		MaxHeaderBytes: config.MaxHeaderBytes,
		ConnState:      stateCallback,
		//TO-DO ErrorLog
	}

	logs.WriteBasic("Golang server started on http://" + connectionString + " [" + time.Now().Truncate(0).String() + "]")
	http.HandleFunc("/", handler.Handle)
	http.HandleFunc("/server-status/", handler.ServerStatus)
	http.HandleFunc("/gop-info/", handler.GopInfo)
	log.Fatal(server.ListenAndServe()) //log.Fatal will tell if we cant bind the port
}
