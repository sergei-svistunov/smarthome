package webserver

import (
	"fmt"
	"html"
	"net/http"
)

type Webserver struct {
	httpServer *http.Server
	serveMux   *http.ServeMux
	DoneChan   chan bool
}

func NewWebserver(addr string) (*Webserver, error) {
	webserver := new(Webserver)
	webserver.DoneChan = make(chan bool)
	webserver.serveMux = http.NewServeMux()
	webserver.httpServer = &http.Server{
		Addr:    addr,
		Handler: webserver.serveMux,
	}

	webserver.serveMux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	go func() {
		webserver.httpServer.ListenAndServe()
		webserver.DoneChan <- true
	}()

	return webserver, nil
}
