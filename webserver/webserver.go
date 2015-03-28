package webserver

import (
	"fmt"
	"github.com/sergei-svistunov/smarthome/x10"
	"golang.org/x/net/websocket"
	"net/http"
)

type Webserver struct {
	httpServer    *http.Server
	serveMux      *http.ServeMux
	DoneChan      chan bool
	x10Controller *x10.Controller
}

func NewWebserver(addr string, x10Controller *x10.Controller) (*Webserver, error) {
	webserver := new(Webserver)
	webserver.DoneChan = make(chan bool)
	webserver.serveMux = http.NewServeMux()
	webserver.httpServer = &http.Server{
		Addr:    addr,
		Handler: webserver.serveMux,
	}
	webserver.x10Controller = x10Controller

	webserver.serveMux.Handle("/", http.FileServer(http.Dir("html")))

	wsDevices := NewWsDevices(x10Controller)
	webserver.serveMux.Handle("/devices/", websocket.Handler(func(ws *websocket.Conn) {
		wsDevices.AssignConnection(ws)
	}))

	go func() {
		err := webserver.httpServer.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
		webserver.DoneChan <- true
	}()

	return webserver, nil
}
